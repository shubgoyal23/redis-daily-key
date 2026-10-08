#!/bin/sh
# busybox crond does not pass the container's environment to jobs, so
# snapshot the relevant variables into a wrapper the cron line calls.
set -eu

{
	echo '#!/bin/sh'
	echo 'cd /app'
	env | grep -E '^(REDIS_|TZ=)' | sed "s/'/'\\\\''/g; s/=\(.*\)/='\1'/; s/^/export /"
	echo 'export GOMAXPROCS=1 GOGC=off'
	echo 'exec /app/redis-daily'
} > /app/run.sh
chmod 700 /app/run.sh

# Equivalent of systemd's Persistent=true: optionally run once at startup.
if [ "${RUN_ON_START:-false}" = "true" ]; then
	/app/run.sh || echo "startup run failed (will retry at next schedule)"
fi

exec crond -f -l 8
