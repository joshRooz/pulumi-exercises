#!/bin/bash

if [[ $(grep -wci "ubuntu" /etc/os-release) -gt 0 ]] ; then
  apt-get update -y && apt-get upgrade -y
  apt-get install -y nginx

  echo "Hello Pulumi from ${HOSTNAME}" | sudo tee -a /var/www/html/index.html
else
  yum install --assumeyes yum-utils

  cat <<-EOF > /etc/yum.repos.d/nginx.repo
[nginx-stable]
name=nginx stable repo
baseurl=http://nginx.org/packages/centos/\$releasever/\$basearch/
gpgcheck=1
enabled=1
gpgkey=https://nginx.org/keys/nginx_signing.key
module_hotfixes=true
EOF

  yum install --assumeyes nginx
  echo "Hello Pulumi from ${HOSTNAME}" | sudo tee /usr/share/nginx/html/index.html
  systemctl start nginx
fi


exit 0
