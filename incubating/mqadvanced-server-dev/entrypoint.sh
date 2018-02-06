#!/bin/bash

# Change admin password
if [ -n "${MQ_ADMIN_PASSWORD}" ]; then
    echo admin:${MQ_ADMIN_PASSWORD} | chpasswd
fi
# Change app password
if [ -n "${MQ_APP_PASSWORD}" ]; then
    echo app:${MQ_APP_PASSWORD} | chpasswd
fi

# Delete the MQSC with developer defaults, if requested
if [ "${MQ_DEV}" != "true" ]; then
    rm -f /etc/mqm/dev.mqsc
fi

exec runmqserver