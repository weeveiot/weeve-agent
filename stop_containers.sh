#! /bin/sh

#! temp file for dev
docker stop feel-like-m_001.weevenetwork_mqtt-ingress_latest.0
docker stop feel-like-m_001.weevenetwork_fluctuation-filter_latest.1
docker stop feel-like-m_001.weevenetwork_comparison-filter_latest.2 
docker stop feel-like-m_001.weevenetwork_slack-alert_latest.3
docker system prune
rm ./known_manifests.jsonl