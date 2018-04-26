/* eslint-disable */

// See: https://github.com/rancher/rancher/issues/1874#issuecomment-180082502

var request = require("request");
var child_process = require("child_process");
var ci = require("./teamcity");
var _ = require("lodash");

function notify(err) {
  var msTeamsChannel = process.env.MSTEAMS_DEPLOY_CHANNEL;
  var name = process.env.TEAMCITY_PROJECT_NAME;
  var version = `${process.env.MAJOR_VERSION}.${process.env.MINOR_VERSION}`;
  var jiraURL = process.env.JIRA_URL;

  if (!msTeamsChannel) {
    ci.log(
      "If you set the MSTEAMS_DEPLOY_CHANNEL env parameter we will notify when the upgrade is complete."
    );
    return;
  }

  var branch = child_process.execSync("git rev-parse --abbrev-ref HEAD");

  child_process.exec(
    'git log --pretty=format:"%h - %an, %ar : %s" --graph --since=1.week',
    function(err, stdout, stderr) {
      if (err) {
        ci.fatal(
          "Error Collecting Log Info Teams: " +
            err +
            " (Stderr: " +
            stderr +
            ")"
        );
      }

      var message = "";

      if (jiraURL) {
        var issueIDExp = /([A-Z]{2,5}-[0-9]{0,5})/g;
        var matches = _.uniq(issueIDExp.exec(stdout));

        message =
          "<div><p>The following JIRA issues are related to this deployment:</p>";

        if (matches.length > 0) {
          matches.forEach(function(m) {
            message +=
              '<div><a href="' +
              jiraURL +
              "/browse/" +
              m +
              '">' +
              m +
              "</a></div>";
          });
        }

        message += "</div>";
      }

      var title = `Deployed ${name} (Version: ${version}) to `;
      var color = "48A555";
      message +=
        "<div><p>Commits on this branch in the past 7 days:</p><pre>" +
        stdout +
        "</pre></div>";

      if (err) {
        title =
          "FAILED to Deployed Pipeline-UI (Version: " +
          pkgJson.version +
          ") to ";
        message = "<div>ERROR: " + err + "</div>";
        color = "FF0000";
      }

      if (branch === "master") {
        title += " Production";
      } else {
        title += " Test";
      }

      var deployMessage = {
        title: title,
        text: message,
        themeColor: color
      };

      request(
        msTeamsChannel,
        {
          method: "POST",
          body: JSON.stringify(deployMessage)
        },
        function(err, resp, body) {
          if (err) {
            ci.log("WARNING: Could not notify MS Teams about deploy: " + body);
            return;
          }

          ci.log("Notified MS Teams: " + body);
        }
      );
    }
  );
}

module.exports = {
  notify
};
