// ============================================================
// OT Client — mirrors server-side retain/insert/delete model
// ============================================================

function opBaseLen(op) {
    let n = 0;
    for (const c of op.ops) {
        if (c.retain) n += c.retain;
        else if (c.delete) n += c.delete;
    }
    return n;
}

function opTargetLen(op) {
    let n = 0;
    for (const c of op.ops) {
        if (c.retain) n += c.retain;
        else if (c.insert) n += c.insert.length;
    }
    return n;
}

function applyOp(doc, op) {
    let pos = 0, result = "";
    for (const c of op.ops) {
        if (c.retain) {
            result += doc.slice(pos, pos + c.retain);
            pos += c.retain;
        } else if (c.insert) {
            result += c.insert;
        } else if (c.delete) {
            pos += c.delete;
        }
    }
    return result;
}

// Build operation from CodeMirror change
function makeOp(cm, change) {
    const ops = [];
    const docLen = cm.getValue().length; // length AFTER the change
    const from = cm.indexFromPos(change.from);

    // Compute the length of removed text
    const removedText = change.removed.join("\n");
    const removedLen = removedText.length;

    // Compute the length of inserted text
    const insertedText = change.text.join("\n");
    const insertedLen = insertedText.length;

    // The base (before) doc length
    const baseLen = docLen - insertedLen + removedLen;

    if (from > 0) ops.push({ retain: from });
    if (removedLen > 0) ops.push({ delete: removedLen });
    if (insertedLen > 0) ops.push({ insert: insertedText });
    const remaining = baseLen - from - removedLen;
    if (remaining > 0) ops.push({ retain: remaining });

    return { ops };
}

// Iterator for walking through operation components
class OpIterator {
    constructor(op) {
        this.ops = op.ops || [];
        this.index = 0;
        this.offset = 0;
    }

    hasNext() {
        return this.index < this.ops.length;
    }

    peekType() {
        if (!this.hasNext()) return "none";
        const c = this.ops[this.index];
        if (c.insert) return "insert";
        if (c.delete) return "delete";
        return "retain";
    }

    peekLen() {
        if (!this.hasNext()) return 0;
        const c = this.ops[this.index];
        if (c.retain) return c.retain - this.offset;
        if (c.insert) return c.insert.length - this.offset;
        if (c.delete) return c.delete - this.offset;
        return 0;
    }

    take(n) {
        const c = this.ops[this.index];
        const remaining = this.peekLen();

        if (c.retain !== undefined && c.retain > 0) {
            const len = (n >= remaining) ? remaining : n;
            if (n >= remaining) { this.index++; this.offset = 0; }
            else { this.offset += n; }
            return { retain: len };
        }
        if (c.insert) {
            if (n === 0 || n >= remaining) {
                const s = c.insert.slice(this.offset);
                this.index++; this.offset = 0;
                return { insert: s };
            }
            const s = c.insert.slice(this.offset, this.offset + n);
            this.offset += n;
            return { insert: s };
        }
        if (c.delete !== undefined && c.delete > 0) {
            const len = (n >= remaining) ? remaining : n;
            if (n >= remaining) { this.index++; this.offset = 0; }
            else { this.offset += n; }
            return { delete: len };
        }

        this.index++;
        return {};
    }
}

// Transform two concurrent operations
function transform(a, b) {
    if (opBaseLen(a) !== opBaseLen(b)) {
        throw new Error(`base lengths differ: ${opBaseLen(a)} vs ${opBaseLen(b)}`);
    }

    const ap = [], bp = [];
    const ia = new OpIterator(a), ib = new OpIterator(b);

    while (ia.hasNext() || ib.hasNext()) {
        if (ia.peekType() === "insert" && ib.peekType() === "insert") {
            const c = ia.take(0);
            ap.push(c);
            bp.push({ retain: c.insert.length });
            continue;
        }
        if (ia.peekType() === "insert") {
            const c = ia.take(0);
            ap.push(c);
            bp.push({ retain: c.insert.length });
            continue;
        }
        if (ib.peekType() === "insert") {
            const c = ib.take(0);
            bp.push(c);
            ap.push({ retain: c.insert.length });
            continue;
        }

        if (!ia.hasNext() || !ib.hasNext()) break;

        const n = Math.min(ia.peekLen(), ib.peekLen());
        const ca = ia.take(n);
        const cb = ib.take(n);

        if (ca.retain && cb.retain) {
            ap.push({ retain: n });
            bp.push({ retain: n });
        } else if (ca.delete && cb.retain) {
            ap.push({ delete: n });
        } else if (ca.retain && cb.delete) {
            bp.push({ delete: n });
        }
        // delete vs delete: both drop these chars
    }

    return [{ ops: compact(ap) }, { ops: compact(bp) }];
}

function compact(ops) {
    const result = [];
    for (const c of ops) {
        if (result.length === 0) { result.push(c); continue; }
        const last = result[result.length - 1];
        if (c.retain && last.retain) { last.retain += c.retain; }
        else if (c.delete && last.delete) { last.delete += c.delete; }
        else if (c.insert && last.insert) { last.insert += c.insert; }
        else { result.push(c); }
    }
    return result;
}

// ============================================================
// OT Client State Machine
// ============================================================

// States: synchronized, awaitingAck, awaitingAckWithBuffer
let state = "synchronized";
let pending = null;  // op sent to server, awaiting ack
let buffer = null;   // op buffered while waiting for ack
let revision = 0;
let docId = "";

function sendOp(op) {
    if (!ws || ws.readyState !== WebSocket.OPEN) return;
    ws.send(JSON.stringify({
        type: "op",
        docId: docId,
        revision: revision,
        op: op
    }));
}

function clientSendOp(op) {
    switch (state) {
        case "synchronized":
            pending = op;
            state = "awaitingAck";
            sendOp(op);
            break;
        case "awaitingAck":
            buffer = op;
            state = "awaitingAckWithBuffer";
            break;
        case "awaitingAckWithBuffer":
            // Compose buffer with new op
            buffer = compose(buffer, op);
            break;
    }
}

function handleAck(newRevision) {
    revision = newRevision;
    switch (state) {
        case "awaitingAck":
            pending = null;
            state = "synchronized";
            break;
        case "awaitingAckWithBuffer":
            pending = buffer;
            buffer = null;
            state = "awaitingAck";
            sendOp(pending);
            break;
    }
}

function handleRemoteOp(serverOp) {
    switch (state) {
        case "synchronized":
            applyRemoteOp(serverOp);
            revision++;
            break;
        case "awaitingAck": {
            const [pendingP, serverP] = transform(pending, serverOp);
            pending = pendingP;
            applyRemoteOp(serverP);
            revision++;
            break;
        }
        case "awaitingAckWithBuffer": {
            const [pendingP, serverP1] = transform(pending, serverOp);
            const [bufferP, serverP2] = transform(buffer, serverP1);
            pending = pendingP;
            buffer = bufferP;
            applyRemoteOp(serverP2);
            revision++;
            break;
        }
    }
}

// Compose: apply a then b, return combined operation
function compose(a, b) {
    if (opTargetLen(a) !== opBaseLen(b)) {
        throw new Error("compose: length mismatch");
    }

    const result = [];
    const ia = new OpIterator(a), ib = new OpIterator(b);

    while (ia.hasNext() || ib.hasNext()) {
        // If a is delete, it doesn't produce output, so take from a
        if (ia.peekType() === "delete") {
            result.push(ia.take(0));
            continue;
        }
        // If b is insert, it doesn't consume input, so take from b
        if (ib.peekType() === "insert") {
            result.push(ib.take(0));
            continue;
        }

        if (!ia.hasNext() || !ib.hasNext()) break;

        const n = Math.min(ia.peekLen(), ib.peekLen());
        const ca = ia.take(n);
        const cb = ib.take(n);

        if (ca.retain && cb.retain) {
            result.push({ retain: n });
        } else if (ca.retain && cb.delete) {
            result.push({ delete: n });
        } else if (ca.insert && cb.retain) {
            result.push({ insert: ca.insert });
        } else if (ca.insert && cb.delete) {
            // insert then delete same chars — they cancel out
        }
    }

    return { ops: compact(result) };
}

// ============================================================
// CodeMirror integration
// ============================================================

let editor;
let suppressChange = false;

function initEditor() {
    editor = CodeMirror(document.getElementById("editor"), {
        value: "",
        lineNumbers: true,
        lineWrapping: true,
        theme: "default",
    });

    editor.on("change", (cm, change) => {
        if (suppressChange) return;
        if (change.origin === "setValue") return;

        const op = makeOp(cm, change);
        clientSendOp(op);
    });
}

function applyRemoteOp(op) {
    // Apply to CodeMirror without triggering our change handler
    suppressChange = true;
    const doc = editor.getValue();
    const newDoc = applyOp(doc, op);

    // Preserve cursor position
    const cursor = editor.getCursor();
    const cursorIndex = editor.indexFromPos(cursor);
    const newCursorIndex = transformIndex(cursorIndex, op);

    editor.setValue(newDoc);
    editor.setCursor(editor.posFromIndex(newCursorIndex));
    suppressChange = false;
}

// Transform a cursor index through an operation
function transformIndex(index, op) {
    let pos = 0, newIndex = index;
    for (const c of op.ops) {
        if (pos > index) break;
        if (c.retain) {
            pos += c.retain;
        } else if (c.insert) {
            if (pos <= index) newIndex += c.insert.length;
            pos += 0; // insert doesn't consume input
        } else if (c.delete) {
            if (pos + c.delete <= index) {
                newIndex -= c.delete;
            } else if (pos < index) {
                newIndex -= (index - pos);
            }
            pos += c.delete;
        }
    }
    return Math.max(0, newIndex);
}

// ============================================================
// WebSocket connection
// ============================================================

let ws;
let users = {};
let reconnectTimer;

function connect() {
    const protocol = location.protocol === "https:" ? "wss:" : "ws:";
    ws = new WebSocket(`${protocol}//${location.host}/ws`);

    ws.onopen = () => {
        setStatus(true);
        ws.send(JSON.stringify({ type: "join", docId: docId }));
    };

    ws.onmessage = (event) => {
        const msg = JSON.parse(event.data);
        switch (msg.type) {
            case "doc":
                handleDocMessage(msg);
                break;
            case "ack":
                handleAck(msg.revision);
                break;
            case "op":
                handleRemoteOp(msg.op);
                revision = msg.revision;
                break;
            case "join":
                addUser(msg.clientId, msg.name, msg.color);
                break;
            case "leave":
                removeUser(msg.clientId);
                break;
            case "error":
                console.error("Server error:", msg.message);
                break;
        }
    };

    ws.onclose = () => {
        setStatus(false);
        reconnectTimer = setTimeout(connect, 2000);
    };

    ws.onerror = () => {
        ws.close();
    };
}

function handleDocMessage(msg) {
    revision = msg.revision || 0;
    state = "synchronized";
    pending = null;
    buffer = null;

    suppressChange = true;
    editor.setValue(msg.content || "");
    suppressChange = false;

    users = {};
    if (msg.clients) {
        for (const c of msg.clients) {
            users[c.id] = { name: c.name, color: c.color };
        }
    }
    renderUsers();
}

// ============================================================
// UI helpers
// ============================================================

function setStatus(connected) {
    const dot = document.getElementById("connection-status");
    dot.className = "status-dot " + (connected ? "connected" : "disconnected");
}

function addUser(id, name, color) {
    users[id] = { name, color };
    renderUsers();
}

function removeUser(id) {
    delete users[id];
    renderUsers();
}

function renderUsers() {
    const bar = document.getElementById("users-bar");
    const keys = Object.keys(users);
    document.getElementById("user-count").textContent = keys.length + " user" + (keys.length !== 1 ? "s" : "");

    bar.innerHTML = "";
    for (const id of keys) {
        const u = users[id];
        const tag = document.createElement("span");
        tag.className = "user-tag";
        tag.style.backgroundColor = u.color + "22";
        tag.style.color = u.color;
        tag.innerHTML = `<span class="user-dot" style="background:${u.color}"></span>${u.name}`;
        bar.appendChild(tag);
    }
}

// ============================================================
// Initialization
// ============================================================

function getDocId() {
    let id = location.hash.slice(1);
    if (!id) {
        id = Math.random().toString(36).slice(2, 10);
        location.hash = id;
    }
    return id;
}

document.addEventListener("DOMContentLoaded", () => {
    docId = getDocId();
    document.getElementById("doc-id").textContent = "#" + docId;

    document.getElementById("copy-link").addEventListener("click", () => {
        navigator.clipboard.writeText(location.href);
        const btn = document.getElementById("copy-link");
        btn.textContent = "Copied!";
        setTimeout(() => btn.textContent = "Copy Link", 1500);
    });

    initEditor();
    connect();
});

// Handle hash change (navigate to different doc)
window.addEventListener("hashchange", () => {
    if (ws) ws.close();
    docId = getDocId();
    document.getElementById("doc-id").textContent = "#" + docId;
    state = "synchronized";
    pending = null;
    buffer = null;
    connect();
});
