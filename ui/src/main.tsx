#!/usr/bin/env node
import React from "react";
import {render} from "ink";
import {Command} from "commander";
import {App} from "./app.js";
import {AvmClient} from "./avm-client.js";

const program = new Command();

program
  .name("avm-ui")
  .description("Full-screen Agent VM terminal UI")
  .option("--avm <path>", "path to the avm binary", process.env.AVM_BIN ?? "avm")
  .parse(process.argv);

const options = program.opts<{avm: string}>();

render(<App client={new AvmClient(options.avm)} />, {
  exitOnCtrlC: true
});
