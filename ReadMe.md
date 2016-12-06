## Funktion Operator


This operator watches for `Subscription` resources. When a new Subscription is created then this operator will spin up a matching `Deployment` which consumes from some endpoint and typically invokes a function using HTTP.
 
A `Subscription` is modelled as a Kubernetes `ConfigMap` with the label `kind.funktion.fabric8.io: "Subscription"`. A ConfigMap is used so that the entries inside the `ConfigMap` can be mounted as files inside the `Deployment`. For example this will typically involve storing the `funktion.yml` file or maybe a Spring Boot `application.properties` file inside the `ConfigMap` like [this example subscription](examples/subscription1.yml)

A `Connector` is generated [for every Camel Component](https://github.com/fabric8io/funktion/blob/master/connectors/) and each connector has an associated `ConfigMap` resource like [this example](https://github.com/fabric8io/funktion/blob/master/connectors/connector-timer/src/main/fabric8/timer-cm.yml) which uses the label `kind.funktion.fabric8.io: "Connector"`. The `Connector` stores the `Deployment` metadata, the `schema.yml` for editing the connectors endpoint URL and the `documentation.adoc` documentation for using the Connector.

### Using Funktion Operator

First you need to install the Connectors in your Kubernetes namespace

TODO

Now you can create a subscription using a YAML configuration like this:

   kubectl apply -f https://github.com/fabric8io/funktion-operator/blob/master/examples/subscription1.yml
   
Hopefully we'll have a simple CLI and web UI for doing this soon!   
   
   