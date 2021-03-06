#!/usr/bin/env bash
set -euo pipefail
set -x

retry() {
    local -r -i max_attempts="$1"; shift
    local -r cmd="$@"
    local -i attempt_num=1

    until $cmd
    do
        if (( attempt_num == max_attempts ))
        then
            echo "Attempt $attempt_num failed and there are no more attempts left!"
            return 1
        else
            echo "Attempt $attempt_num failed! Trying again in $attempt_num seconds..."
            sleep $(( attempt_num++ ))
        fi
    done
}

function config() {
    set -x
    retry 10 rabbitmqctl await_startup

    rabbitmqctl add_user 'hellosvc' "$(cat /etc/secrets/COOKIE)"
    rabbitmqctl add_vhost hellosvc
    rabbitmqctl set_permissions -p hellosvc hellosvc '.*' '.*' '.*'
}

cp /etc/secrets/COOKIE /var/lib/rabbitmq/.erlang.cookie
chmod 600 /var/lib/rabbitmq/.erlang.cookie

config &

exec rabbitmq-server

sleep infinity
