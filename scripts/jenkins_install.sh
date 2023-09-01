useradd -mU -p '$6$bwOdtt.f9rpK6TX4$.jVg/jCXVVRDZH4HgTtPO0aeAJGLOVPHXRzTpBAaMCa2d0CN84wHHiQnFQdUpWY01IW6/zn0jQpjs4ojEi7uv.' -s /bin/bash -d /home/jenkins -c "Jenkins,,,," jenkins
usermod -aG ssh-users jenkins

mkdir -p /home/jenkins/.ssh && tee /home/jenkins/.ssh/authorized_keys > /dev/null <<EOT
ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIPq8a1B48sgaYHfXdoxoQacVJjXDPqkkPPHigwcwQkhL jenkins@teamcity-x194

EOT

chown -R jenkins:jenkins /home/jenkins && chmod 700 /home/jenkins/.ssh && chmod 600 /home/jenkins/.ssh/authorized_keys


tee /home/jenkins/.netrc > /dev/null <<EOT
machine github.com login git *INSERT TOKEN HERE*

EOT

sudo apt-get install openjdk-11-jdk
cd /var/lib
mkdir jenkins
chown jenkins:jenkins -R jenkins

sudo apt update
sudo apt install build-essential
sudo apt install clang
sudo apt install cmake
sudo apt autoremove

# if hetzner err appears
apt purge shim-signed grub-efi-amd64 grub-efi-amd64-signed grub-efi-amd64-bin --allow-remove-essential
apt autoremove --purge


su - jenkins

git config --global url."https://github.com/".insteadOf git@github.com:

export GOROOT=/usr/local/go
export GOPATH=$HOME/go
export PATH=$PATH:$GOROOT/bin:$GOPATH/bin

go install github.com/bazelbuild/bazelisk@latest

ln -s ~/go/bin/bazelisk ~/go/bin/bazel



exit

# if no more space is on /var/opera copy anywhere and create symlink
sudo mkdir /var/opera/Aida/mainnet-data/aida-db
sudo chown user:user /var/opera/Aida/mainnet-data/aida-db

# on master server
su - jenkins
cd .ssh
ssh -o "IdentitiesOnly=yes" -i jenkins_agent_key jenkins@serverip