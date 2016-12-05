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

package main

import (
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/fabric8io/funktion-operator/pkg/funktion"

	"github.com/go-kit/kit/log"
	flag "github.com/spf13/pflag"
	"k8s.io/client-go/1.5/tools/clientcmd"
)

var (
	analyticsEnabled bool
)

func Main() int {
	logger := log.NewContext(log.NewLogfmtLogger(os.Stdout)).
		With("ts", log.DefaultTimestampUTC, "caller", log.DefaultCaller).
		With("operator", "funktion")

	flagset := flag.NewFlagSet(os.Args[0], flag.ExitOnError)

	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	flagset.StringVar(&loadingRules.ExplicitPath, "kubeconfig", "", "Path to the config file to use for CLI requests.")

	overrides := &clientcmd.ConfigOverrides{}
	overrideFlags := clientcmd.RecommendedConfigOverrideFlags("")
	clientcmd.BindOverrideFlags(overrides, flagset, overrideFlags)

	kubeConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, overrides)

	flagset.Parse(os.Args[1:])

	cfg, err := kubeConfig.ClientConfig()
	if err != nil {
		logger.Log("msg", "failed to create Kubernetes client config", "error", err)
		return 1
	}

	ko, err := funktion.New(cfg, logger)
	if err != nil {
		logger.Log("error", err)
		return 1
	}

	stopc := make(chan struct{})
	errc := make(chan error)
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		if err := ko.Run(stopc); err != nil {
			errc <- err
		}
		wg.Done()
	}()

	term := make(chan os.Signal)
	signal.Notify(term, os.Interrupt, syscall.SIGTERM)
	select {
	case <-term:
		fmt.Fprintln(os.Stderr)
		logger.Log("msg", "Received SIGTERM, exiting gracefully...")
		close(stopc)
		wg.Wait()
	case err := <-errc:
		logger.Log("msg", "Unexpected error received", "error", err)
		close(stopc)
		wg.Wait()
		return 1
	}

	return 0
}

func main() {
	os.Exit(Main())
}
