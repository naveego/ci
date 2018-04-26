/* eslint-disable */

// See: https://github.com/rancher/rancher/issues/1874#issuecomment-180082502

var request = require("request");
var child_process = require("child_process");
var ci = require("./teamcity");
var msteams = require("./msteams");

function requireEnv(key) {
  let val = process.env[key];
  if (val) {
    return val;
  }
  fatal(`You must set the "${key}" environment variable.`);
}

function fatal(message) {
  ci.buildProblem(message);
  process.exit(1);
}

function upgradeService() {
  var rancherURL = requireEnv("RANCHER_URL");
  var rancherKey = requireEnv("RANCHER_KEY");
  var rancherSecret = requireEnv("RANCHER_SECRET");
  var serviceName = requireEnv("RANCHER_SERVICE");
  var imageName = requireEnv("RANCHER_IMAGE");

  if (!imageName.startsWith("docker:")) {
    imageName = "docker:" + imageName;
  }

  ci.log("Fetching current service settings from " + rancherURL + " ...");

  var serviceWithoutStack = serviceName.split("/")[1];

  request(
    rancherURL + "/v1/services?name=" + serviceWithoutStack,
    {
      auth: {
        username: rancherKey,
        password: rancherSecret
      }
    },
    function(err, resp, body) {
      if (err) {
        ci.log("ERROR: Could not get current settings from rancher: " + err);
      }

      ci.log("Successfully fetched service settings.");

      var services = JSON.parse(body);
      var service = services.data[0];

      console.log('service settings', service);
      
      if (service.state !== "active") {
        fatal(
          "ERROR: Unexpected service state. Expected 'active' but was '" +
          service.state +
          '".'
        );
      }
      
      if (service.type !== "service") {
        fatal(
          "ERROR: Unexpected service type. Expected 'service' but was '" +
          service.type +
          "'."
        );
      }
      
      
      console.log('upgrading to image', imageName);
      
      var upgrade = service.upgrade;
      upgrade.launchConfig = service.launchConfig;
      upgrade.secondaryLaunchConfigs = service.secondaryLaunchConfigs;
      upgrade.toServiceStrategy = {};

      upgrade.launchConfig.imageUuid = imageName;
      upgrade.inServiceStrategy.launchConfig.imageUuid = imageName;

      
      console.log('upgrade settings', upgrade);
      
      ci.progress("Starting upgrade of " + serviceName);
      request(
        service.actions.upgrade,
        {
          method: "POST",
          auth: {
            username: rancherKey,
            password: rancherSecret
          },
          body: JSON.stringify(upgrade)
        },
        function(err, resp, body) {
          if (err) {
            fatal("ERROR: Could not start upgrade: " + err);
          }

          var startData = JSON.parse(body);

          if (startData.type === "error") {
            fatal(
              "ERROR: Could not start upgrade: " + startData.code + "\n" + body
            );
          }

          ci.progress("Upgrade started successfully, waiting to confirm...");
          waitForUpgrade(service, rancherKey, rancherSecret)
            .then(function(s) {
              ci.progress("Finishing Upgrade...");

              request(
                s.actions.finishupgrade,
                {
                  method: "POST",
                  auth: {
                    username: rancherKey,
                    password: rancherSecret
                  }
                },
                function(err, resp) {
                  if (err) {
                    return fatal("Could not finish upgrade: " + err);
                  }

                  msteams.notify();
                  ci.progress("Upgrade Succesfull!");
                }
              );
            })
            .catch(function(err) {
              ci.progress("Rolling Back Upgrade...");
              request(
                s.actions.rollback,
                {
                  method: "POST",
                  auth: {
                    username: rancherKey,
                    password: rancherSecret
                  }
                },
                function(err, resp) {
                  if (err) {
                    msteams.notify(err);
                    return fatal("Could not finish upgrade: " + err);
                  }

                  ci.log("Rollback succesfull!");
                }
              );
              msteams.notify(err);
              fatal(err);
            });
        }
      );
    }
  );
}

function waitForUpgrade(service, rancherKey, rancherSecret) {
  return new Promise(function(resolve, reject) {
    var count = 0;
    var loop = setInterval(function() {
      request(
        service.links.self,
        {
          auth: {
            username: rancherKey,
            password: rancherSecret
          }
        },
        function(err, resp, body) {
          count++;
          if (err) {
            reject(err);
            clearInterval(loop);
          }
          var s = JSON.parse(body);
          ci.log("Service Status: " + s.state + "...");

          if (s.state === "upgraded") {
            resolve(s);
            clearInterval(loop);
          } else if (s.state !== "upgrading") {
            reject("Failed to complete upgrade. Rancher status was " + s.state);
            clearInterval(loop);
          }
        }
      );
    }, 1000);
  });
}

module.exports = {
  upgradeService
};
