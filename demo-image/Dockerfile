FROM gcr.io/spiffe-io/spire-agent:1.0.1
ADD date /
ENTRYPOINT ["/usr/bin/dumb-init", "/opt/spire/bin/spire-agent","api","fetch","x509","-socketPath","/sock/agent.sock"]
