#!/bin/bash

# Get OS information using /etc/os-release
if [ -f /etc/os-release ]; then
    echo "OS Information:" > /tmp/os_info.txt
    cat /etc/os-release >> /tmp/os_info.txt
else
    echo "OS information file not found!" > /tmp/os_info.txt
fi

# Append Kernel information
echo -e "\nKernel Information:" >> /tmp/os_info.txt
uname -r >> /tmp/os_info.txt

# Append CPU information
echo -e "\nCPU Information:" >> /tmp/os_info.txt
lscpu >> /tmp/os_info.txt

# Append Memory information
echo -e "\nMemory Information:" >> /tmp/os_info.txt
free -h >> /tmp/os_info.txt

