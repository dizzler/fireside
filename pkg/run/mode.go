package run

import (
    log "github.com/sirupsen/logrus"

    "fireside/pkg/configure"
    "fireside/pkg/envoy/xds"
    "fireside/pkg/pipeline/events"
)

// run FireSide in 'ca' mode
func RunModeCa(config *configure.Config) {
    log.Debug("running RunModeCa function")
    log.Info("creating TLS Trust Domains for Envoy TLS 'secrets'")
    for _, policy := range config.Policies {
        _, err := xds.MakeTlsTrustDomains(policy.EnvoyPolicy.Secrets)
        if err != nil {
            log.WithError(err).Fatal("failure encountered while creating TLS Trust Domain")
        }
    }
    return
}

// run FireSide in 'server' mode
func RunModeServer(config *configure.Config) {
    log.Debug("running RunModeServer function")
    // create a data processing pipeline for events
    go events.CreateEventsPipelines(config)

    // serve dynamic resource configurations to Envoy nodes
    go xds.ServeEnvoyXds(config)

    cc := make(chan struct{})
    <-cc

    return
}
