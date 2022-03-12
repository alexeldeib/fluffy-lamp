#!/usr/bin/env bash
set -euo pipefail

set -x

# Retries a command on failure.
# $1 - the max number of attempts
# $2... - the command to run
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

cp /etc/secrets/COOKIE /var/lib/rabbitmq/.erlang.cookie
chmod 600 /var/lib/rabbitmq/.erlang.cookie

# sleep infinity

retry 10 rabbitmqctl await_startup -n rabbit@rabbitmq-0.rabbitmq-headless.default.svc.cluster.local
retry 10 rabbitmqctl authenticate_user 'hellosvc' "$(cat /etc/secrets/COOKIE)" -n rabbit@rabbitmq-0.rabbitmq-headless.default.svc.cluster.local
/workspace/bin/app
