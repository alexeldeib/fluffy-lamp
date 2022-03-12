#!/usr/bin/env bash
set -euo pipefail
set -x

function config() {
    rabbitmqctl await_startup

    rabbitmqctl add_user 'hellosvc' "$(cat /etc/secrets/COOKIE)"
    rabbitmqctl add_vhost hellosvc
    rabbitmqctl set_permissions -p hellosvc hellosvc '.*' '.*' '.*'
}

cp /etc/secrets/COOKIE /var/lib/rabbitmq/.erlang.cookie
chmod 600 /var/lib/rabbitmq/.erlang.cookie

config &

exec rabbitmq-server

sleep infinity
