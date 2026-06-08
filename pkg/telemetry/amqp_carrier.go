package telemetry

import (
	amqp "github.com/rabbitmq/amqp091-go"
	"go.opentelemetry.io/otel/propagation"
)

// AMQPHeadersCarrier  implements the propagation.TextMapCarrier interface for AMQP headers.
type AMQPHeadersCarrier struct {
	Headers amqp.Table
}

var _ propagation.TextMapCarrier = (*AMQPHeadersCarrier)(nil)

func (c AMQPHeadersCarrier) Get(key string) string {
	if v, ok := c.Headers[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return  ""
}

func (c AMQPHeadersCarrier) Keys() []string {
	keys := make([]string, 0, len(c.Headers))
	for k := range c.Headers {
		keys = append(keys, k)
	}
	return keys
}

func (c AMQPHeadersCarrier) Set(key string, value string) {
	c.Headers[key] = value
}
