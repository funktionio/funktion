## Funktion Operator

This operator watches for `Subscription` resources. When a new Subscription is created then this operator will spin up a matching `Deployment` which consumes from some endpoint and typically invokes a function using HTTP.
 
A `Subscription` is modelled as a Kubenretes `ConfigMap` with the label `funktion.fabric8.io/kind: "Subscription"`. A ConfigMap is used so that the entries inside the `ConfigMap` can be mounted as files inside the `Deployment`. For example this will typically involve storing the `funktion.yml` file or maybe a Spring Boot `application.properties` file inside the `ConfigMap` like []this example subscription](examples/subscription1.yml)

