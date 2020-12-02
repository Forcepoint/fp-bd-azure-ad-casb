#!/bin/bash

echo "install Python3.."
sudo yum install -y https://centos7.iuscommunity.org/ius-release.rpm
sudo yum install -y python36u python36u-libs python36u-devel python36u-pip

echo "installing Golang.."
sudo yum install wget -y
wget https://dl.google.com/go/go1.14.1.linux-amd64.tar.gz
sudo tar -zxvf go1.14.1.linux-amd64.tar.gz -C /usr/local
echo "export GOROOT=/usr/local/go" | sudo tee -a /etc/profile
echo "export PATH=$PATH:/usr/local/go/bin" | sudo tee -a /etc/profile
source /etc/profile
echo "installing Azure Cli..."
sudo rpm --import https://packages.microsoft.com/keys/microsoft.asc
sudo sh -c 'echo -e "[azure-cli]
name=Azure CLI
baseurl=https://packages.microsoft.com/yumrepos/azure-cli
enabled=1
gpgcheck=1
gpgkey=https://packages.microsoft.com/keys/microsoft.asc" > /etc/yum.repos.d/azure-cli.repo'
sudo yum install azure-cli -y
mkdir /var/azure_casb
chmod +x azure_casb
mv ./azure_casb /var/azure_casb/
mv ./azure_casb.yml /var/azure_casb/
mv ./azure_casb.service /etc/systemd/system/
sudo systemctl enable azure_casb.service

