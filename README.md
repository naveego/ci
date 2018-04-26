# README #

This is the [Drone](http://docs.drone.io/installation/) CI server configuration repo.

* Github CI: https://gci.naveego.com
* Bitbucket CI: https://bci.naveego.com

When starting the containers you must provide environment variables. The easiest way is to create a `.env` file.

### Github ENV:
```
DRONE_GITHUB_CLIENT={obtained from github oath config}
DRONE_GITHUB_SECRET={obtained from github oath config}
DRONE_SECRET={a long random string}
DRONE_HOST={the full URL to the drone server, like https://gci.naveego.com}
```

### Bitbucket ENV:
```
DRONE_BITBUCKET_CLIENT={obtained from github oath config}
DRONE_BITBUCKET_SECRET={obtained from github oath config}
DRONE_SECRET={a long random string}
DRONE_HOST={the full URL to the drone server, like https://gci.naveego.com}
```

# DroneCfg

This repo also provides a configuration tool (`dronecfg`) which can automate the addition of repositories to a Drone instance, or recreate all repositories if an instance gets broken.