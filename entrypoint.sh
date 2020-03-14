#!/bin/bash

./json_exporter --port "${PORT}" "${ENDPOINT_JSON}" /opt/json_exporter/config.yml
