#!/bin/sh

docker run \
-p 389:389 \
-p 636:636 \
-e LDAP_ORGANISATION="TFO Bozen" \
-e LDAP_DOMAIN="example.org" \
--rm -it --name ldap osixia/openldap:1.5.0