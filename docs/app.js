(function () {
  "use strict";

  var el = document.getElementById("terminal-output");
  var replayBtn = document.getElementById("terminal-replay");
  if (!el) return;

  var sequences = [
    { type: "line", html: '<span class="prompt">$</span> <span class="cmd">go install github.com/mantyx-io/m2a/cmd/m2a@latest</span>' },
    { type: "pause", ms: 400 },
    { type: "line", html: '<span class="dim">…</span> <span class="ok">done</span>' },
    { type: "pause", ms: 500 },
    { type: "line", html: "" },
    { type: "line", html: '<span class="prompt">$</span> <span class="cmd">m2a https://your-agent.example.com</span>' },
    { type: "pause", ms: 600 },
    { type: "line", html: '<span class="dim">GET</span> <span class="label">/.well-known/agent-card.json</span>' },
    { type: "pause", ms: 450 },
    { type: "line", html: '<span class="ok">OK</span> <span class="label">Example Agent</span> <span class="dim">·</span> <span class="label">HTTP+JSON</span>' },
    { type: "pause", ms: 500 },
    { type: "line", html: '<span class="dim">──</span>' },
    { type: "pause", ms: 300 },
    { type: "line", html: '<span class="user">You:</span> What is A2A?' },
    { type: "pause", ms: 700 },
    {
      type: "line",
      html:
        '<span class="agent">Agent:</span> A protocol for agents to discover each other and send messages over HTTP or JSON-RPC.',
    },
    { type: "pause", ms: 400 },
    { type: "line", html: '<span class="dim">──</span>' },
    { type: "line", html: '<span class="dim">Enter</span> · <span class="dim">Esc</span>' },
  ];

  var cancelled = false;
  var runId = 0;

  function sleep(ms) {
    return new Promise(function (resolve) {
      setTimeout(resolve, ms);
    });
  }

  async function runSequence(id) {
    el.innerHTML = "";
    for (var i = 0; i < sequences.length; i++) {
      if (id !== runId || cancelled) return;
      var item = sequences[i];
      if (item.type === "pause") {
        await sleep(item.ms);
        continue;
      }
      if (item.type === "line") {
        var line = document.createElement("div");
        line.innerHTML = item.html;
        el.appendChild(line);
        el.scrollTop = el.scrollHeight;
        await sleep(item.html === "" ? 120 : 35);
      }
    }
    if (id !== runId || cancelled) return;
    var cursor = document.createElement("span");
    cursor.className = "terminal-cursor";
    cursor.setAttribute("aria-hidden", "true");
    el.appendChild(cursor);
  }

  function start() {
    runId++;
    cancelled = false;
    runSequence(runId);
  }

  if (replayBtn) {
    replayBtn.addEventListener("click", function () {
      start();
    });
  }

  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", start);
  } else {
    start();
  }
})();
