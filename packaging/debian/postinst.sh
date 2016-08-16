#!/bin/sh
set -e

# Install kerberos dependency
pip install kerberos
pip install xmltodict
pip install requests_kerberos
pip install pymongo