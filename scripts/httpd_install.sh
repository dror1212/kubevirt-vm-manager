#!/bin/bash
# Bash script to install and run httpd on port 80

# Install httpd
yum -y install httpd

# Start httpd
systemctl start httpd

# Enable httpd to start on boot
systemctl enable httpd

# Open port 80 in the firewall (if applicable)
firewall-cmd --permanent --add-port=80/tcp
firewall-cmd --reload
