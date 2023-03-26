#!/bin/zsh

HOST="localhost"
PORT="389"

URL=$(ldapurl -h "$HOST" -p "$PORT")

ADMIN_BIND_DN="cn=admin,dc=example,dc=org"

echo "Base url is: $URL"

search() {
  # Search for all entries
  # -x: simple authentication
  # -H: LDAP URL
  # -b: base DN to search
  # -D: bind DN
  # -w: bind password
  ldapsearch -x -H "$URL" -b "dc=example,dc=org" -D "$ADMIN_BIND_DN" -w admin
}

read -r

search

read -r

# Add a new person
# -f: LDIF file for input
ldapadd -x -H "$URL" -D "$ADMIN_BIND_DN" -w admin -f ldif/add.ldif

read -r

search

read -r

# Modify a person
ldapmodify -x -H "$URL" -D "$ADMIN_BIND_DN" -w admin -f ldif/modify.ldif

read -r

search

read -r

# Delete a person
ldapdelete -x -H "$URL" -D "$ADMIN_BIND_DN" -w admin "cn=John Xina,dc=example,dc=org"
echo "Deleted John Xina"

read -r

search