FROM dindbase

run apt-get update && \
    apt-get install -y openssh-server && \
    rm -rf /var/lib/apt/lists/* /var/cache/*
expose 2376 3376 22
volume /var/lib/docker
run touch /var/log/auth.log
run echo "root:root" | chpasswd
run sed -i 's/PermitRootLogin without-password/PermitRootLogin yes/' /etc/ssh/sshd_config

# SSH login fix. Otherwise user is kicked off after login
run sed 's@session\s*required\s*pam_loginuid.so@session optional pam_loginuid.so@g' -i /etc/pam.d/sshd
add daemons.sh /daemons.sh
cmd ["/daemons.sh"]
