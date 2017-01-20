# <img src="https://raw.githubusercontent.com/funktionio/funktion-connectors/master/docs/images/icon.png" width="24" height="24"/>&nbsp; Funktion

**Funktion** is an open source event driven lambda style programming model on top of [Kubernetes](http://kubernetes.io). This project provides a command line tool for working with `Funktion`

Funktion supports hundreds of different [trigger endpoint URLs](http://camel.apache.org/components.html) including most network protocols, transports, databases, messaging systems, social networks, cloud services and SaaS offerings.

In a sense funktion is a [serverless](https://www.quora.com/What-is-Serverless-Computing) approach to event driven microservices as you focus on just writing _funktions_ and Kubernetes takes care of the rest. Its not that there's no servers; its more that you as the funktion developer don't have to worry about managing them.

<p align="center">
  <a href="http://fabric8.io/">
  	<img src="https://raw.githubusercontent.com/funktionio/funktion-connectors/master/docs/images/icon.png" alt="funktion logo" width="200" height="200"/>
  </a>
</p>


### Getting Started

Please [Install Funktion](https://funktion.fabric8.io/docs/#install) then follow the [Getting Started Guide](https://funktion.fabric8.io/docs/#get-started) 

### Documentation

Please see [the website](https://funktion.fabric8.io/) and the [User Guide](https://funktion.fabric8.io/docs/) 


### License

This project is [Apache Licensed](license.md)

### Building

You will need a recent install of `go` along with `glide`.

Then the first time you want to build you need to do this:

```
mkdir -p $GOROOT/funktionio/funktion
cd $GOROOT/funktionio
git clone https://github.com/funktionio/funktion.git
cd funktion
make bootstrap
```

Now whenever you want to do build you can just type

```
make
```

and you'll get a `./funktion` binary you can play with

#### Running locally outside of docker

If you want to hack on the `operator` its often easier to just run it locally on your laptop using your local build via

```
./funktion operate
```

And scale down/delete the `funktion-operator` thats running inside kubernetes. 

Provided your machine can talk to your kubernetes cluster via:

```
kubectl get pod
kubectl get node
```

then the `funktion` binary should be able to monitor and operate all your flows and functions.
