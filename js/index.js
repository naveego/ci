#!/usr/bin/env node

const rancher = require("./rancher");
const ci = require("./teamcity");
const argv = require("yargs").argv;

function execute() {
  let command = argv._[0];

  switch (command) {
    case "deploy":
      ci.progress("Running deploy!");
      return rancher.upgradeService();
    default:
      ci.buildProblem(`Unrecognized command "${command}"`);
  }
}

execute();
