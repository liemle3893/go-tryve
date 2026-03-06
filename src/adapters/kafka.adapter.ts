/**
 * E2E Test Runner - Kafka Adapter
 *
 * Apache Kafka operations using kafkajs
 */

import { AdapterError, AssertionError, TimeoutError } from '../errors';
import { runAssertion, type BaseAssertion } from '../assertions/assertion-runner';
import type { AdapterConfig, AdapterContext, AdapterStepResult, Logger } from '../types';
import { BaseAdapter } from './base.adapter';

export class KafkaAdapter extends BaseAdapter {
  private kafka: import('kafkajs').Kafka | null = null;
  private producer: import('kafkajs').Producer | null = null;

  constructor(config: AdapterConfig, logger: Logger) {
    super(config, logger);
  }

  get name(): string {
    return 'kafka';
  }

  async connect(): Promise<void> {
    if (this.connected) return;

    try {
      const { Kafka } = await import('kafkajs');

      const brokers = Array.isArray(this.config.brokers)
        ? this.config.brokers
        : [this.config.brokers as string];

      this.kafka = new Kafka({
        clientId: (this.config.clientId as string) || 'e2e-runner',
        brokers,
        ssl: this.config.ssl as boolean | undefined,
        sasl: this.config.sasl as import('kafkajs').SASLOptions | undefined,
        connectionTimeout: (this.config.connectionTimeout as number) || 10000,
        requestTimeout: (this.config.requestTimeout as number) || 30000,
      });

      this.producer = this.kafka.producer();
      await this.producer.connect();

      this.connected = true;
      this.logger.info('Kafka connected');
    } catch (error) {
      throw new AdapterError(
        'kafka',
        'connect',
        `Failed to connect: ${error instanceof Error ? error.message : String(error)}`
      );
    }
  }

  async disconnect(): Promise<void> {
    if (this.producer) {
      await this.producer.disconnect();
      this.producer = null;
    }
    this.connected = false;
    this.logger.info('Kafka disconnected');
  }

  async execute(
    action: string,
    params: Record<string, unknown>,
    ctx: AdapterContext
  ): Promise<AdapterStepResult> {
    if (!this.producer) {
      throw new AdapterError('kafka', action, 'Not connected');
    }

    this.logAction(action, { topic: params.topic });
    const start = Date.now();

    try {
      let result: AdapterStepResult;

      switch (action) {
        case 'produce':
          result = await this.produceMessage(params);
          break;
        case 'consume':
          result = await this.consumeMessages(params);
          break;
        case 'waitFor':
          result = await this.waitForMessage(params, ctx);
          break;
        case 'clear':
          result = this.successResult(null, Date.now() - start);
          break;
        default:
          throw new AdapterError('kafka', action, `Unknown action: ${action}`);
      }

      this.logResult(action, true, result.duration);
      return result;
    } catch (error) {
      const duration = Date.now() - start;
      this.logResult(action, false, duration);

      if (
        error instanceof AssertionError ||
        error instanceof AdapterError ||
        error instanceof TimeoutError
      ) {
        throw error;
      }

      throw new AdapterError(
        'kafka',
        action,
        error instanceof Error ? error.message : String(error)
      );
    }
  }

  async healthCheck(): Promise<boolean> {
    return this.connected;
  }

  private async produceMessage(params: Record<string, unknown>): Promise<AdapterStepResult> {
    const start = Date.now();
    const topic = params.topic as string;
    const messages: import('kafkajs').Message[] = [];
    const inputMessages = params.messages || [params.message];

    for (const msg of inputMessages as Record<string, unknown>[]) {
      messages.push({
        key: msg.key as string | undefined,
        value: JSON.stringify(msg.value || msg),
        partition: msg.partition as number | undefined,
        headers: msg.headers as Record<string, string> | undefined,
      });
    }

    await this.producer!.send({ topic, messages });
    return this.successResult({ count: messages.length }, Date.now() - start);
  }

  private async consumeMessages(params: Record<string, unknown>): Promise<AdapterStepResult> {
    const start = Date.now();
    const topic = params.topic as string;
    const count = (params.count as number) || 1;
    const timeout = (params.timeout as number) || 10000;
    const messages: unknown[] = [];
    let resolved = false;

    const consumer = this.kafka!.consumer({
      groupId: `e2e-runner-consumer-${Date.now()}-${Math.random().toString(36).slice(2, 8)}`,
    });
    await consumer.connect();
    await consumer.subscribe({ topic, fromBeginning: false });

    try {
      return await new Promise<AdapterStepResult>((resolve, reject) => {
        const timeoutId = setTimeout(async () => {
          if (resolved) return;
          resolved = true;
          await consumer.disconnect();
          resolve(this.successResult(messages, Date.now() - start));
        }, timeout);

        consumer.run({
          eachMessage: async ({ message }) => {
            try {
              if (resolved) return;
              const value = JSON.parse(message.value?.toString() || '{}');
              messages.push(value);

              if (messages.length >= count) {
                if (resolved) return;
                resolved = true;
                clearTimeout(timeoutId);
                await consumer.disconnect();
                resolve(this.successResult(messages, Date.now() - start));
              }
            } catch (error) {
              if (resolved) return;
              resolved = true;
              clearTimeout(timeoutId);
              reject(error);
            }
          },
        }).catch((error) => {
          if (resolved) return;
          resolved = true;
          clearTimeout(timeoutId);
          reject(error);
        });
      });
    } catch (error) {
      await consumer.disconnect().catch(() => {});
      throw error;
    }
  }

  private async waitForMessage(
    params: Record<string, unknown>,
    ctx: AdapterContext
  ): Promise<AdapterStepResult> {
    const start = Date.now();
    const topic = params.topic as string;
    const timeout = (params.timeout as number) || 30000;
    const filter = params.filter as Record<string, unknown>;
    let resolved = false;

    const consumer = this.kafka!.consumer({
      groupId: `e2e-runner-consumer-${Date.now()}-${Math.random().toString(36).slice(2, 8)}`,
    });
    await consumer.connect();
    await consumer.subscribe({ topic, fromBeginning: false });

    try {
      return await new Promise<AdapterStepResult>((resolve, reject) => {
        const timeoutId = setTimeout(async () => {
          if (resolved) return;
          resolved = true;
          await consumer.disconnect();
          resolve(
            this.failResult(
              new TimeoutError(
                `Waiting for message matching filter: ${JSON.stringify(filter)}`,
                timeout
              ),
              Date.now() - start
            )
          );
        }, timeout);

        consumer.run({
          eachMessage: async ({ message }) => {
            try {
              if (resolved) return;
              const body = JSON.parse(message.value?.toString() || '{}');

              if (this.matchesFilter(body, filter)) {
                if (resolved) return;
                resolved = true;
                clearTimeout(timeoutId);
                await consumer.disconnect();

                if (params.capture) {
                  for (const [varName, path] of Object.entries(params.capture as Record<string, string>)) {
                    ctx.capture(varName, this.getNestedValue(body, path));
                  }
                }

                if (params.assert) {
                  const assertions = Array.isArray(params.assert) ? params.assert : [params.assert];
                  for (const assertion of assertions as BaseAssertion[]) {
                    runAssertion(body, assertion);
                  }
                }

                resolve(this.successResult(body, Date.now() - start));
              }
            } catch (error) {
              if (resolved) return;
              resolved = true;
              clearTimeout(timeoutId);
              reject(error);
            }
          },
        }).catch((error) => {
          if (resolved) return;
          resolved = true;
          clearTimeout(timeoutId);
          reject(error);
        });
      });
    } catch (error) {
      await consumer.disconnect().catch(() => {});
      throw error;
    }
  }

  private matchesFilter(message: Record<string, unknown>, filter: Record<string, unknown>): boolean {
    if (!filter) return true;
    for (const [key, expected] of Object.entries(filter)) {
      const actual = this.getNestedValue(message, key);
      if (actual !== expected) return false;
    }
    return true;
  }

  private getNestedValue(obj: unknown, path: string): unknown {
    if (!path) return obj;
    return path.split('.').reduce((current, key) => {
      if (current && typeof current === 'object') {
        return (current as Record<string, unknown>)[key];
      }
      return undefined;
    }, obj);
  }

}
