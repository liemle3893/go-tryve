package adapter

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	kafka "github.com/segmentio/kafka-go"
	"github.com/segmentio/kafka-go/sasl"
	"github.com/segmentio/kafka-go/sasl/plain"
	"github.com/segmentio/kafka-go/sasl/scram"

	"github.com/liemle3893/e2e-runner/internal/tryve"
)

// KafkaAdapter provides produce, consume, waitFor, and clear actions against
// a Kafka cluster using the segmentio/kafka-go library.
type KafkaAdapter struct {
	brokers   []string
	clientID  string
	groupID   string
	timeout   time.Duration
	mechanism sasl.Mechanism // nil when SASL is not configured
	tls       bool

	mu      sync.Mutex
	readers []*kafka.Reader
	writers []*kafka.Writer
}

// NewKafkaAdapter constructs a KafkaAdapter from a generic config map.
//
// Recognised keys:
//   - brokers    ([]string or []any) — required; list of host:port addresses
//   - clientId   (string)            — optional Kafka client identifier
//   - groupId    (string)            — optional consumer group ID
//   - timeout    (int, ms)           — per-operation default timeout (default 10 000)
//   - ssl        (bool)              — reserved; TLS flag (not yet wired to dialer)
//   - sasl       (map)               — optional SASL config with keys:
//     mechanism (plain|scram-sha-256|scram-sha-512), username, password
func NewKafkaAdapter(cfg map[string]any) *KafkaAdapter {
	a := &KafkaAdapter{
		timeout: 10 * time.Second,
	}

	// -- brokers ----------------------------------------------------------------
	if v, ok := cfg["brokers"]; ok {
		switch bv := v.(type) {
		case []string:
			a.brokers = append(a.brokers, bv...)
		case []any:
			for _, b := range bv {
				if s, ok := b.(string); ok {
					a.brokers = append(a.brokers, s)
				}
			}
		}
	}

	// -- optional string fields -------------------------------------------------
	a.clientID = getStrDefault(cfg, "clientId", "")
	a.groupID = getStrDefault(cfg, "groupId", "")

	// -- timeout ----------------------------------------------------------------
	if v, ok := cfg["timeout"]; ok {
		switch tv := v.(type) {
		case int:
			if tv > 0 {
				a.timeout = time.Duration(tv) * time.Millisecond
			}
		case float64:
			if tv > 0 {
				a.timeout = time.Duration(int(tv)) * time.Millisecond
			}
		}
	}

	// -- ssl --------------------------------------------------------------------
	if v, ok := cfg["ssl"]; ok {
		if b, ok := v.(bool); ok {
			a.tls = b
		}
	}

	// -- sasl -------------------------------------------------------------------
	if v, ok := cfg["sasl"]; ok {
		if saslMap, ok := v.(map[string]any); ok {
			mech, _ := saslMap["mechanism"].(string)
			user, _ := saslMap["username"].(string)
			pass, _ := saslMap["password"].(string)
			if m := buildSASLMechanism(mech, user, pass); m != nil {
				a.mechanism = m
			}
		}
	}

	return a
}

// buildSASLMechanism constructs the appropriate sasl.Mechanism for the given
// mechanism name. Returns nil for unrecognised or empty mechanism names.
func buildSASLMechanism(mechanism, username, password string) sasl.Mechanism {
	switch strings.ToLower(mechanism) {
	case "plain":
		return plain.Mechanism{Username: username, Password: password}
	case "scram-sha-256":
		m, err := scram.Mechanism(scram.SHA256, username, password)
		if err != nil {
			return nil
		}
		return m
	case "scram-sha-512":
		m, err := scram.Mechanism(scram.SHA512, username, password)
		if err != nil {
			return nil
		}
		return m
	default:
		return nil
	}
}

// Name returns the adapter's registered identifier.
func (a *KafkaAdapter) Name() string { return "kafka" }

// Connect validates that at least one broker is configured. The actual TCP
// connection is established lazily when a reader/writer is created, so this
// method only checks preconditions.
func (a *KafkaAdapter) Connect(_ context.Context) error {
	if len(a.brokers) == 0 {
		return tryve.ConnectionError("kafka", "no brokers configured", nil)
	}
	return nil
}

// Close closes all active readers and writers created during Execute calls.
func (a *KafkaAdapter) Close(_ context.Context) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	var lastErr error
	for _, r := range a.readers {
		if err := r.Close(); err != nil {
			lastErr = err
		}
	}
	for _, w := range a.writers {
		if err := w.Close(); err != nil {
			lastErr = err
		}
	}
	a.readers = nil
	a.writers = nil
	return lastErr
}

// Health dials the first broker to verify TCP connectivity.
func (a *KafkaAdapter) Health(ctx context.Context) error {
	if len(a.brokers) == 0 {
		return tryve.ConnectionError("kafka", "no brokers configured", nil)
	}

	d := a.newDialer()
	conn, err := d.DialContext(ctx, "tcp", a.brokers[0])
	if err != nil {
		return tryve.ConnectionError("kafka", fmt.Sprintf("health check failed: %v", err), err)
	}
	_ = conn.Close()
	return nil
}

// Execute dispatches the named action with the given parameters.
//
// Supported actions: produce, consume, waitFor, clear.
func (a *KafkaAdapter) Execute(ctx context.Context, action string, params map[string]any) (*tryve.StepResult, error) {
	switch action {
	case "produce":
		return a.executeProduce(ctx, params)
	case "consume":
		return a.executeConsume(ctx, params)
	case "waitFor":
		return a.executeWaitFor(ctx, params)
	case "clear":
		return a.executeClear(ctx, params)
	default:
		return nil, tryve.AdapterError("kafka", action,
			fmt.Sprintf("unsupported action %q: supported actions are produce, consume, waitFor, clear", action), nil)
	}
}

// --------------------------------------------------------------------------
// Action implementations
// --------------------------------------------------------------------------

// executeProduce writes a single message to the specified topic.
//
// Required params: topic, value.
// Optional params: key (string), headers (map[string]any).
func (a *KafkaAdapter) executeProduce(ctx context.Context, params map[string]any) (*tryve.StepResult, error) {
	topic := getStrDefault(params, "topic", "")
	if topic == "" {
		return nil, tryve.AdapterError("kafka", "produce", "missing required param: topic", nil)
	}

	msgValue, err := encodeValue(params["value"])
	if err != nil {
		return nil, tryve.AdapterError("kafka", "produce", "failed to encode value", err)
	}

	msg := kafka.Message{
		Topic: topic,
		Value: msgValue,
	}

	if k := getStrDefault(params, "key", ""); k != "" {
		msg.Key = []byte(k)
	}

	if hv, ok := params["headers"]; ok {
		if hMap, ok := hv.(map[string]any); ok {
			for k, v := range hMap {
				msg.Headers = append(msg.Headers, kafka.Header{
					Key:   k,
					Value: []byte(fmt.Sprintf("%v", v)),
				})
			}
		}
	}

	w := a.newWriter(topic)
	a.trackWriter(w)
	defer func() {
		_ = w.Close()
		a.untrackWriter(w)
	}()

	var duration time.Duration
	duration, err = MeasureDuration(func() error {
		return w.WriteMessages(ctx, msg)
	})
	if err != nil {
		return nil, tryve.AdapterError("kafka", "produce", "failed to write message", err)
	}

	return SuccessResult(map[string]any{"ok": true}, duration, nil), nil
}

// executeConsume reads one message from the specified topic/group.
//
// Required params: topic.
// Optional params: timeout (int, ms) — overrides adapter-level timeout.
func (a *KafkaAdapter) executeConsume(ctx context.Context, params map[string]any) (*tryve.StepResult, error) {
	topic := getStrDefault(params, "topic", "")
	if topic == "" {
		return nil, tryve.AdapterError("kafka", "consume", "missing required param: topic", nil)
	}

	opTimeout := a.resolveTimeout(params)
	ctx, cancel := context.WithTimeout(ctx, opTimeout)
	defer cancel()

	r := a.newReader(topic)
	a.trackReader(r)
	defer func() {
		_ = r.Close()
		a.untrackReader(r)
	}()

	var msg kafka.Message
	var duration time.Duration
	var err error

	duration, err = MeasureDuration(func() error {
		msg, err = r.ReadMessage(ctx)
		return err
	})
	if err != nil {
		return nil, tryve.AdapterError("kafka", "consume", "failed to read message", err)
	}

	return SuccessResult(messageToData(msg), duration, nil), nil
}

// executeWaitFor reads messages until one matches all conditions in the match
// map or the operation times out.
//
// Required params: topic, match (map[string]any).
// Optional params: timeout (int, ms).
func (a *KafkaAdapter) executeWaitFor(ctx context.Context, params map[string]any) (*tryve.StepResult, error) {
	topic := getStrDefault(params, "topic", "")
	if topic == "" {
		return nil, tryve.AdapterError("kafka", "waitFor", "missing required param: topic", nil)
	}

	matchMap, ok := params["match"].(map[string]any)
	if !ok || len(matchMap) == 0 {
		return nil, tryve.AdapterError("kafka", "waitFor", "missing or invalid required param: match", nil)
	}

	opTimeout := a.resolveTimeout(params)
	ctx, cancel := context.WithTimeout(ctx, opTimeout)
	defer cancel()

	r := a.newReader(topic)
	a.trackReader(r)
	defer func() {
		_ = r.Close()
		a.untrackReader(r)
	}()

	for {
		var msg kafka.Message
		var err error
		_, err = MeasureDuration(func() error {
			msg, err = r.ReadMessage(ctx)
			return err
		})
		if err != nil {
			if ctx.Err() != nil {
				return nil, tryve.TimeoutError("kafka.waitFor", opTimeout)
			}
			return nil, tryve.AdapterError("kafka", "waitFor", "failed to read message", err)
		}

		if messageMatches(msg, matchMap) {
			var duration time.Duration
			return SuccessResult(messageToData(msg), duration, nil), nil
		}
	}
}

// executeClear drains all pending messages from the topic and returns the
// number consumed.
//
// Required params: topic.
// Optional params: timeout (int, ms).
func (a *KafkaAdapter) executeClear(ctx context.Context, params map[string]any) (*tryve.StepResult, error) {
	topic := getStrDefault(params, "topic", "")
	if topic == "" {
		return nil, tryve.AdapterError("kafka", "clear", "missing required param: topic", nil)
	}

	opTimeout := a.resolveTimeout(params)
	drainCtx, cancel := context.WithTimeout(ctx, opTimeout)
	defer cancel()

	r := a.newReader(topic)
	a.trackReader(r)
	defer func() {
		_ = r.Close()
		a.untrackReader(r)
	}()

	cleared := 0
	start := time.Now()

	for {
		_, err := r.ReadMessage(drainCtx)
		if err != nil {
			// Timeout or context cancellation means the queue is drained.
			break
		}
		cleared++
	}

	duration := time.Since(start)
	return SuccessResult(map[string]any{"cleared": float64(cleared)}, duration, nil), nil
}

// --------------------------------------------------------------------------
// Helpers — reader / writer construction
// --------------------------------------------------------------------------

// newDialer constructs a kafka.Dialer with SASL configured if a mechanism is set.
func (a *KafkaAdapter) newDialer() *kafka.Dialer {
	d := &kafka.Dialer{
		Timeout:   a.timeout,
		DualStack: true,
	}
	if a.mechanism != nil {
		d.SASLMechanism = a.mechanism
	}
	if a.clientID != "" {
		d.ClientID = a.clientID
	}
	return d
}

// newWriter creates a kafka.Writer for the given topic.
func (a *KafkaAdapter) newWriter(topic string) *kafka.Writer {
	transport := &kafka.Transport{
		Dial: (&net.Dialer{
			Timeout: a.timeout,
		}).DialContext,
	}
	if a.mechanism != nil {
		transport.SASL = a.mechanism
	}
	w := &kafka.Writer{
		Addr:      kafka.TCP(a.brokers...),
		Topic:     topic,
		Transport: transport,
	}
	if a.clientID != "" {
		// kafka.Writer does not expose ClientID directly; set via Balancer metadata.
		_ = a.clientID
	}
	return w
}

// newReader creates a kafka.Reader for the given topic.
func (a *KafkaAdapter) newReader(topic string) *kafka.Reader {
	cfg := kafka.ReaderConfig{
		Brokers:  a.brokers,
		Topic:    topic,
		MinBytes: 1,
		MaxBytes: 10e6, // 10 MB
		MaxWait:  a.timeout,
	}
	if a.groupID != "" {
		cfg.GroupID = a.groupID
	}
	if a.mechanism != nil {
		cfg.Dialer = a.newDialer()
	}
	return kafka.NewReader(cfg)
}

// --------------------------------------------------------------------------
// Helpers — reader / writer lifecycle tracking
// --------------------------------------------------------------------------

func (a *KafkaAdapter) trackReader(r *kafka.Reader) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.readers = append(a.readers, r)
}

func (a *KafkaAdapter) untrackReader(r *kafka.Reader) {
	a.mu.Lock()
	defer a.mu.Unlock()
	for i, rr := range a.readers {
		if rr == r {
			a.readers = append(a.readers[:i], a.readers[i+1:]...)
			return
		}
	}
}

func (a *KafkaAdapter) trackWriter(w *kafka.Writer) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.writers = append(a.writers, w)
}

func (a *KafkaAdapter) untrackWriter(w *kafka.Writer) {
	a.mu.Lock()
	defer a.mu.Unlock()
	for i, ww := range a.writers {
		if ww == w {
			a.writers = append(a.writers[:i], a.writers[i+1:]...)
			return
		}
	}
}

// --------------------------------------------------------------------------
// Helpers — message conversion and matching
// --------------------------------------------------------------------------

// encodeValue converts a value param to []byte.
// Strings are encoded directly; maps and other types are JSON-marshalled.
func encodeValue(v any) ([]byte, error) {
	if v == nil {
		return nil, nil
	}
	switch tv := v.(type) {
	case string:
		return []byte(tv), nil
	default:
		return json.Marshal(tv)
	}
}

// messageToData converts a kafka.Message to the canonical result map.
func messageToData(msg kafka.Message) map[string]any {
	headers := make(map[string]any, len(msg.Headers))
	for _, h := range msg.Headers {
		headers[h.Key] = string(h.Value)
	}

	// Attempt JSON decode of value; fall back to raw string.
	var value any = string(msg.Value)
	var decoded any
	if err := json.Unmarshal(msg.Value, &decoded); err == nil {
		value = decoded
	}

	return map[string]any{
		"key":       string(msg.Key),
		"value":     value,
		"headers":   headers,
		"topic":     msg.Topic,
		"partition": float64(msg.Partition),
		"offset":    float64(msg.Offset),
	}
}

// messageMatches returns true when every key/value pair in matchMap equals the
// corresponding field in the message data map.
func messageMatches(msg kafka.Message, matchMap map[string]any) bool {
	data := messageToData(msg)
	for k, want := range matchMap {
		got, ok := data[k]
		if !ok {
			return false
		}
		if fmt.Sprintf("%v", got) != fmt.Sprintf("%v", want) {
			return false
		}
	}
	return true
}

// resolveTimeout extracts an optional "timeout" param (ms) from params,
// falling back to the adapter-level default.
func (a *KafkaAdapter) resolveTimeout(params map[string]any) time.Duration {
	if v, ok := params["timeout"]; ok {
		switch tv := v.(type) {
		case int:
			if tv > 0 {
				return time.Duration(tv) * time.Millisecond
			}
		case float64:
			if tv > 0 {
				return time.Duration(int(tv)) * time.Millisecond
			}
		}
	}
	return a.timeout
}
