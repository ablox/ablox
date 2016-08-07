# ablox

[![Join the chat at https://gitter.im/ablox/ablox](https://badges.gitter.im/ablox/ablox.svg)](https://gitter.im/ablox/ablox?utm_source=badge&utm_medium=badge&utm_campaign=pr-badge&utm_content=badge)

This system is currenly a work in progress. The server side is working and is easy to setup. It currenly only supports the NBD protocol.

To use:

1. Clone the repository

2. Run from prompt> go get github.com/urfave/cli

3. Run from prompt> go run server/server.go hostname port
  * hostname - the hostname or IP address you want to listen on. Hostname can be localhost if you only want it available locally
  * port - The port you want to listen on.

Put any files you want to attach in a subfolder "sample_disks". They are immediately avaiable for all users.

Right now, any users are allowed to accesss any files. File locking is not occuring on the server so you must make sure you only modify the disk from one system. Multiple read-only copies are fine.

Please file issues if you find any problems.

### Background info
* https://tour.golang.org/welcome/1
* http://www.thegeekstuff.com/2009/02/nbd-tutorial-network-block-device-jumpstart-guide/
* https://docs.docker.com/docker-for-mac/
* https://hub.docker.com/_/golang/
* https://www.minio.io/
* http://www.microhowto.info/howto/connect_to_a_remote_block_device_using_nbd.html

### Installation

* Go
* QEMU
```sh
brew install qemu
```

## Setup Docker and Ubuntu under VMWare

sudo su -
apt-get update
apt-get install open-ssl openssh-server open-vm-tools apt-transport-https ca-certificates nbd-client nbd-serverß

Follow instructions: https://docs.docker.com/engine/installation/linux/ubuntulinux/

// to be able to connect to an NBD server, you have to make sure the module is loaded. On starting the OS, run:
sudo modprobe nbd



	### Tools
* Wireshark
* QEMU
* Virtualbox
* Intellij/PyCharm with golang plugin (Delve debugger currently has a bug itself!, so breakpoint will only stop 1 time! and it won't stop anymore even after you restart the debugging session)
