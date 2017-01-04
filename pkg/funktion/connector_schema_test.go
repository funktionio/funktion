//  Copyright 2016 Red Hat, Inc.
//
//  Licensed under the Apache License, Version 2.0 (the "License");
//  you may not use this file except in compliance with the License.
//  You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
//  Unless required by applicable law or agreed to in writing, software
//  distributed under the License is distributed on an "AS IS" BASIS,
//  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//  See the License for the specific language governing permissions and
//  limitations under the License.

package funktion

import (
	"bytes"
	"fmt"
	"runtime"
	"testing"

	"github.com/funktionio/funktion/pkg/spec"
)

const (
	sampleSchemaYaml = `---
component:
  kind: component
  scheme: twitter
  syntax: twitter:kind
  title: Twitter
  description: This component integrates with Twitter to send tweets or search for tweets and more.
  label: api,social
  deprecated: false
  async: false
  javaType: org.apache.camel.component.twitter.TwitterComponent
  groupId: org.apache.camel
  artifactId: camel-twitter
  version: 2.18.1
componentProperties:
  accessToken:
    kind: property
    type: string
    javaType: java.lang.String
    deprecated: false
    secret: false
    description: The access token
  accessTokenSecret:
    kind: property
    type: string
    javaType: java.lang.String
    deprecated: false
    secret: false
    description: The access token secret
  consumerKey:
    kind: property
    type: string
    javaType: java.lang.String
    deprecated: false
    secret: false
    description: The consumer key
  consumerSecret:
    kind: property
    type: string
    javaType: java.lang.String
    deprecated: false
    secret: false
    description: The consumer secret
  httpProxyHost:
    kind: property
    type: string
    javaType: java.lang.String
    deprecated: false
    secret: false
    description: The http proxy host which can be used for the camel-twitter.
  httpProxyUser:
    kind: property
    type: string
    javaType: java.lang.String
    deprecated: false
    secret: false
    description: The http proxy user which can be used for the camel-twitter.
  httpProxyPassword:
    kind: property
    type: string
    javaType: java.lang.String
    deprecated: false
    secret: false
    description: The http proxy password which can be used for the camel-twitter.
  httpProxyPort:
    kind: property
    type: integer
    javaType: int
    deprecated: false
    secret: false
    description: The http proxy port which can be used for the camel-twitter.
properties:
  kind:
    kind: path
    group: common
    required: true
    type: string
    javaType: java.lang.String
    enum:
    - directmessage
    - search
    - streaming/filter
    - streaming/sample
    - streaming/user
    - timeline/home
    - timeline/mentions
    - timeline/retweetsofme
    - timeline/user
    deprecated: false
    secret: false
    description: What polling mode to use direct polling or event based. The event mode is only supported when the endpoint kind is event based.
  accessToken:
    kind: parameter
    group: common
    type: string
    javaType: java.lang.String
    deprecated: false
    secret: false
    description: The access token. Can also be configured on the TwitterComponent level instead.
  accessTokenSecret:
    kind: parameter
    group: common
    type: string
    javaType: java.lang.String
    deprecated: false
    secret: false
    description: The access secret. Can also be configured on the TwitterComponent level instead.
  consumerKey:
    kind: parameter
    group: common
    type: string
    javaType: java.lang.String
    deprecated: false
    secret: false
    description: The consumer key. Can also be configured on the TwitterComponent level instead.
  consumerSecret:
    kind: parameter
    group: common
    type: string
    javaType: java.lang.String
    deprecated: false
    secret: false
    description: The consumer secret. Can also be configured on the TwitterComponent level instead.
  user:
    kind: parameter
    group: common
    type: string
    javaType: java.lang.String
    deprecated: false
    secret: false
    description: Username used for user timeline consumption direct message production etc.
  bridgeErrorHandler:
    kind: parameter
    group: consumer
    label: consumer
    type: boolean
    javaType: boolean
    optionalPrefix: consumer.
    deprecated: false
    secret: false
    defaultValue: false
    description: Allows for bridging the consumer to the Camel routing Error Handler which mean any exceptions occurred while the consumer is trying to pickup incoming messages or the likes will now be processed as a message and handled by the routing Error Handler. By default the consumer will use the org.apache.camel.spi.ExceptionHandler to deal with exceptions that will be logged at WARN/ERROR level and ignored.
  sendEmptyMessageWhenIdle:
    kind: parameter
    group: consumer
    label: consumer
    type: boolean
    javaType: boolean
    optionalPrefix: consumer.
    deprecated: false
    secret: false
    defaultValue: false
    description: If the polling consumer did not poll any files you can enable this option to send an empty message (no body) instead.
  type:
    kind: parameter
    group: consumer
    label: consumer
    type: string
    javaType: org.apache.camel.component.twitter.data.EndpointType
    enum:
    - polling
    - direct
    - event
    deprecated: false
    secret: false
    defaultValue: direct
    description: Endpoint type to use. Only streaming supports event type.
  distanceMetric:
    kind: parameter
    group: consumer (advanced)
    label: consumer,advanced
    type: string
    javaType: java.lang.String
    enum:
    - km
    - mi
    deprecated: false
    secret: false
    defaultValue: km
    description: 'Used by the non-stream geography search to search by radius using
      the configured metrics. The unit can either be mi for miles or km for kilometers.
      You need to configure all the following options: longitude latitude radius and
      distanceMetric.'
  exceptionHandler:
    kind: parameter
    group: consumer (advanced)
    label: consumer,advanced
    type: object
    javaType: org.apache.camel.spi.ExceptionHandler
    optionalPrefix: consumer.
    deprecated: false
    secret: false
    description: To let the consumer use a custom ExceptionHandler. Notice if the option bridgeErrorHandler is enabled then this options is not in use. By default the consumer will deal with exceptions that will be logged at WARN/ERROR level and ignored.
  exchangePattern:
    kind: parameter
    group: consumer (advanced)
    label: consumer,advanced
    type: string
    javaType: org.apache.camel.ExchangePattern
    enum:
    - InOnly
    - RobustInOnly
    - InOut
    - InOptionalOut
    - OutOnly
    - RobustOutOnly
    - OutIn
    - OutOptionalIn
    deprecated: false
    secret: false
    description: Sets the exchange pattern when the consumer creates an exchange.
  latitude:
    kind: parameter
    group: consumer (advanced)
    label: consumer,advanced
    type: number
    javaType: java.lang.Double
    deprecated: false
    secret: false
    description: 'Used by the non-stream geography search to search by latitude. You
      need to configure all the following options: longitude latitude radius and distanceMetric.'
  locations:
    kind: parameter
    group: consumer (advanced)
    label: consumer,advanced
    type: string
    javaType: java.lang.String
    deprecated: false
    secret: false
    description: Bounding boxes created by pairs of lat/lons. Can be used for streaming/filter. A pair is defined as latlon. And multiple paris can be separated by semi colon.
  longitude:
    kind: parameter
    group: consumer (advanced)
    label: consumer,advanced
    type: number
    javaType: java.lang.Double
    deprecated: false
    secret: false
    description: 'Used by the non-stream geography search to search by longitude.
      You need to configure all the following options: longitude latitude radius and
      distanceMetric.'
  pollStrategy:
    kind: parameter
    group: consumer (advanced)
    label: consumer,advanced
    type: object
    javaType: org.apache.camel.spi.PollingConsumerPollStrategy
    optionalPrefix: consumer.
    deprecated: false
    secret: false
    description: A pluggable org.apache.camel.PollingConsumerPollingStrategy allowing you to provide your custom implementation to control error handling usually occurred during the poll operation before an Exchange have been created and being routed in Camel.
  radius:
    kind: parameter
    group: consumer (advanced)
    label: consumer,advanced
    type: number
    javaType: java.lang.Double
    deprecated: false
    secret: false
    description: 'Used by the non-stream geography search to search by radius. You
      need to configure all the following options: longitude latitude radius and distanceMetric.'
  twitterStream:
    kind: parameter
    group: consumer (advanced)
    label: consumer,advanced
    type: object
    javaType: twitter4j.TwitterStream
    deprecated: false
    secret: false
    description: To use a custom instance of TwitterStream
  synchronous:
    kind: parameter
    group: advanced
    label: advanced
    type: boolean
    javaType: boolean
    deprecated: false
    secret: false
    defaultValue: false
    description: Sets whether synchronous processing should be strictly used or Camel is allowed to use asynchronous processing (if supported).
  count:
    kind: parameter
    group: filter
    label: consumer,filter
    type: integer
    javaType: java.lang.Integer
    deprecated: false
    secret: false
    description: Limiting number of results per page.
  filterOld:
    kind: parameter
    group: filter
    label: consumer,filter
    type: boolean
    javaType: boolean
    deprecated: false
    secret: false
    defaultValue: true
    description: Filter out old tweets that has previously been polled. This state is stored in memory only and based on last tweet id.
  keywords:
    kind: parameter
    group: filter
    label: consumer,filter
    type: string
    javaType: java.lang.String
    deprecated: false
    secret: false
    description: Can be used for search and streaming/filter. Multiple values can be separated with comma.
  lang:
    kind: parameter
    group: filter
    label: consumer,filter
    type: string
    javaType: java.lang.String
    deprecated: false
    secret: false
    description: The lang string ISO_639-1 which will be used for searching
  numberOfPages:
    kind: parameter
    group: filter
    label: consumer,filter
    type: integer
    javaType: java.lang.Integer
    deprecated: false
    secret: false
    defaultValue: "1"
    description: The number of pages result which you want camel-twitter to consume.
  sinceId:
    kind: parameter
    group: filter
    label: consumer,filter
    type: integer
    javaType: long
    deprecated: false
    secret: false
    defaultValue: "1"
    description: The last tweet id which will be used for pulling the tweets. It is useful when the camel route is restarted after a long running.
  userIds:
    kind: parameter
    group: filter
    label: consumer,filter
    type: string
    javaType: java.lang.String
    deprecated: false
    secret: false
    description: To filter by user ids for streaming/filter. Multiple values can be separated by comma.
  backoffErrorThreshold:
    kind: parameter
    group: scheduler
    label: consumer,scheduler
    type: integer
    javaType: int
    optionalPrefix: consumer.
    deprecated: false
    secret: false
    description: The number of subsequent error polls (failed due some error) that should happen before the backoffMultipler should kick-in.
  backoffIdleThreshold:
    kind: parameter
    group: scheduler
    label: consumer,scheduler
    type: integer
    javaType: int
    optionalPrefix: consumer.
    deprecated: false
    secret: false
    description: The number of subsequent idle polls that should happen before the backoffMultipler should kick-in.
  backoffMultiplier:
    kind: parameter
    group: scheduler
    label: consumer,scheduler
    type: integer
    javaType: int
    optionalPrefix: consumer.
    deprecated: false
    secret: false
    description: To let the scheduled polling consumer backoff if there has been a number of subsequent idles/errors in a row. The multiplier is then the number of polls that will be skipped before the next actual attempt is happening again. When this option is in use then backoffIdleThreshold and/or backoffErrorThreshold must also be configured.
  delay:
    kind: parameter
    group: scheduler
    label: consumer,scheduler
    type: integer
    javaType: long
    optionalPrefix: consumer.
    deprecated: false
    secret: false
    defaultValue: "60000"
    description: Milliseconds before the next poll.
  greedy:
    kind: parameter
    group: scheduler
    label: consumer,scheduler
    type: boolean
    javaType: boolean
    optionalPrefix: consumer.
    deprecated: false
    secret: false
    defaultValue: false
    description: If greedy is enabled then the ScheduledPollConsumer will run immediately again if the previous run polled 1 or more messages.
  initialDelay:
    kind: parameter
    group: scheduler
    label: consumer,scheduler
    type: integer
    javaType: long
    optionalPrefix: consumer.
    deprecated: false
    secret: false
    defaultValue: "1000"
    description: Milliseconds before the first poll starts. You can also specify time values using units such as 60s (60 seconds) 5m30s (5 minutes and 30 seconds) and 1h (1 hour).
  runLoggingLevel:
    kind: parameter
    group: scheduler
    label: consumer,scheduler
    type: string
    javaType: org.apache.camel.LoggingLevel
    enum:
    - TRACE
    - DEBUG
    - INFO
    - WARN
    - ERROR
    - OFF
    optionalPrefix: consumer.
    deprecated: false
    secret: false
    defaultValue: TRACE
    description: The consumer logs a start/complete log line when it polls. This option allows you to configure the logging level for that.
  scheduledExecutorService:
    kind: parameter
    group: scheduler
    label: consumer,scheduler
    type: object
    javaType: java.util.concurrent.ScheduledExecutorService
    optionalPrefix: consumer.
    deprecated: false
    secret: false
    description: Allows for configuring a custom/shared thread pool to use for the consumer. By default each consumer has its own single threaded thread pool.
  scheduler:
    kind: parameter
    group: scheduler
    label: consumer,scheduler
    type: string
    javaType: org.apache.camel.spi.ScheduledPollConsumerScheduler
    enum:
    - none
    - spring
    - quartz2
    optionalPrefix: consumer.
    deprecated: false
    secret: false
    defaultValue: none
    description: To use a cron scheduler from either camel-spring or camel-quartz2 component
  schedulerProperties:
    kind: parameter
    group: scheduler
    label: consumer,scheduler
    type: object
    javaType: java.util.Map<java.lang.String,java.lang.Object>
    prefix: scheduler.
    multiValue: true
    deprecated: false
    secret: false
    description: To configure additional properties when using a custom scheduler or any of the Quartz2 Spring based scheduler.
  startScheduler:
    kind: parameter
    group: scheduler
    label: consumer,scheduler
    type: boolean
    javaType: boolean
    optionalPrefix: consumer.
    deprecated: false
    secret: false
    defaultValue: true
    description: Whether the scheduler should be auto started.
  timeUnit:
    kind: parameter
    group: scheduler
    label: consumer,scheduler
    type: string
    javaType: java.util.concurrent.TimeUnit
    enum:
    - NANOSECONDS
    - MICROSECONDS
    - MILLISECONDS
    - SECONDS
    - MINUTES
    - HOURS
    - DAYS
    optionalPrefix: consumer.
    deprecated: false
    secret: false
    defaultValue: MILLISECONDS
    description: Time unit for initialDelay and delay options.
  useFixedDelay:
    kind: parameter
    group: scheduler
    label: consumer,scheduler
    type: boolean
    javaType: boolean
    optionalPrefix: consumer.
    deprecated: false
    secret: false
    defaultValue: true
    description: Controls if fixed delay or fixed rate is used. See ScheduledExecutorService in JDK for details.
  httpProxyHost:
    kind: parameter
    group: proxy
    label: proxy
    type: string
    javaType: java.lang.String
    deprecated: false
    secret: false
    description: The http proxy host which can be used for the camel-twitter. Can also be configured on the TwitterComponent level instead.
  httpProxyPassword:
    kind: parameter
    group: proxy
    label: proxy
    type: string
    javaType: java.lang.String
    deprecated: false
    secret: false
    description: The http proxy password which can be used for the camel-twitter. Can also be configured on the TwitterComponent level instead.
  httpProxyPort:
    kind: parameter
    group: proxy
    label: proxy
    type: integer
    javaType: java.lang.Integer
    deprecated: false
    secret: false
    description: The http proxy port which can be used for the camel-twitter. Can also be configured on the TwitterComponent level instead.
  httpProxyUser:
    kind: parameter
    group: proxy
    label: proxy
    type: string
    javaType: java.lang.String
    deprecated: false
    secret: false
    description: The http proxy user which can be used for the camel-twitter. Can also be configured on the TwitterComponent level instead.`
)

func TestLoadConnectorSchema(t *testing.T) {
	schema, err := LoadConnectorSchema([]byte(sampleSchemaYaml))
	if err != nil {
		t.Errorf("Failed to parse YAML %v", err)
		t.Fail()
	}
	if schema == nil {
		t.Errorf("No Schema returned!")
		t.Fail()
	}
	assertEquals(t, schema.Component.Kind, "component")
	assertEquals(t, schema.Component.Scheme, "twitter")

	componentProperties := schema.ComponentProperties
	if len(componentProperties) == 0 {
		t.Errorf("No ComponentProperties returned!")
		t.Fail()
	}
	properties := schema.Properties
	if len(properties) == 0 {
		t.Errorf("No Properties returned!")
		t.Fail()
	}

	logProperties := false
	if logProperties {
		printProperties("Component Properties", schema.ComponentProperties)
		printProperties("Properties", schema.Properties)
	}
}

func printProperties(name string, properties map[string]spec.PropertySpec) {
	fmt.Printf("%s\n", name)
	fmt.Printf("--------------------------------\n")

	for k, p := range properties {
		fmt.Printf("  %s %s %s : %s\n", k, p.Kind, p.Label, p.Description)
	}
	fmt.Println()
}

func assertEquals(t *testing.T, found, expected string) {
	if found != expected {
		logErr(t, found, expected)
	}
}

func logErr(t *testing.T, found, expected string) {
	out := new(bytes.Buffer)

	_, _, line, ok := runtime.Caller(2)
	if ok {
		fmt.Fprintf(out, "Line: %d ", line)
	}
	fmt.Fprintf(out, "Unexpected response.\nExpecting to contain: \n %q\nGot:\n %q\n", expected, found)
	t.Errorf(out.String())
}
