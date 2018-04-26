function runningOnTeamCity() {
  return !!process.env.TEAMCITY_VERSION;
}

// ciMessage writes a message out in a format TeamCity can understand.
// The data parameter can be a string or a map[string]string.
function message(messageType, data) {
  if (runningOnTeamCity()) {
    let message = "##teamcity[" + messageType;

    if (typeof data === "string") {
      let escaped = escape(data);
      message += ` '${escaped}'`;
    } else {
      for (let k of Object.keys(data)) {
        let escaped = escape(data[k]);
        message += ` ${k}='${escaped}'`;
      }
    }
    message += "]";
    console.log(message);
  } else {
    console.log(messageType, data);
  }
}

function escape(s) {
  return s.replace(/(['\n\r\|\]\[])/g, "|$1");
}

// ciBuildProblem reports an error in a way TeamCity can understand.
// If err is nil, this method does nothing.
// This method returns the error to support chaining.
function buildProblem(err) {
  if (err) {
    message("buildProblem", {
      description: err.toString()
    });
  }
  return err;
}

function fatal(err) {
  message("buildProblem", {
    description: err.toString()
  });
  process.exit(1);
}

function progress(msg) {
  message("progressMessage", msg);
}

function log(msg) {
  console.log(msg);
}

module.exports = {
  runningOnTeamCity,
  buildProblem,
  progress,
  message,
  log,
  fatal
};
