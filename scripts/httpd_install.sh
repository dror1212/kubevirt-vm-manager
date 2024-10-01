# Install httpd
sudo yum install -y httpd

# Create a simple index.html file
sudo echo "<html><body><h1>Hello from RHEL HTTP Server!</h1></body></html>" > /var/www/html/index.html

# Set correct permissions for the document root
sudo chmod -R 755 /var/www/html
sudo chown -R apache:apache /var/www/html

# Allow HTTPD to access the /var/www/html directory with SELinux
sudo setsebool -P httpd_read_user_content 1
sudo chcon -R -t httpd_sys_content_t /var/www/html

# Start and enable httpd service
sudo systemctl stop firewalld
sudo systemctl start httpd
sudo systemctl enable httpd
