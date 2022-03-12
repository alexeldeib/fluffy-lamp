FROM rabbitmq:3.9.13-management

# Update system (clear security vulns)
RUN apt-get -y -qq update \
 && apt-get -y -qq upgrade \
 && apt-get -y -qq clean \
 && rm -rf /var/lib/apt/lists/* /tmp/* /var/tmp/*

# Prometheus: https://github.com/rabbitmq/rabbitmq-prometheus
# K8s peer discovery: https://github.com/rabbitmq/rabbitmq-peer-discovery-k8s
# Management: https://github.com/rabbitmq/rabbitmq-management
# HTTP Auth: https://github.com/rabbitmq/rabbitmq-auth-backend-http
# Cache auth: https://github.com/rabbitmq/rabbitmq-auth-backend-cache

RUN rabbitmq-plugins --offline enable rabbitmq_peer_discovery_k8s \
 && rabbitmq-plugins --offline enable rabbitmq_prometheus \
 && rabbitmq-plugins --offline enable rabbitmq_management \
 && rabbitmq-plugins --offline enable rabbitmq_auth_backend_http \
 && rabbitmq-plugins --offline enable rabbitmq_auth_backend_cache \
 && rabbitmq-plugins --offline enable rabbitmq_federation \
 && rabbitmq-plugins --offline enable rabbitmq_federation_management

COPY entrypoint.sh /entrypoint.sh

ENTRYPOINT ["/bin/bash", "/entrypoint.sh"]
