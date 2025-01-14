const WS_URL = 'ws://';
const WS_RECONNECT_INTERVAL = 5000;

let wsInterval;
let ws;

(function() {
    initWebSocket();
})();

function initWebSocket() {
    ws = new WebSocket(WS_URL);
    ws.onopen = function() {
        console.log('websocket connection established');
        clearInterval(wsInterval);
        wsInterval = undefined;
    }
    ws.onclose = function() {
        console.log('websocket connection closed');
        if (!wsInterval) {
            wsInterval = setInterval(function() {
                console.log('attempting to reestablish websocket connection');
                initWebSocket();
            }, WS_RECONNECT_INTERVAL);
        }
    }
    ws.onerror = function(err) {
        if (ws.readyState !== WebSocket.CLOSED) {
            console.error('websocket error', err);
        }
    }
    ws.onmessage = function(evt) {
        if (!evt.data) {
            console.log(' - [ping] websocket message received');
            return;
        }
        console.log(' - [reload] websocket message received: ' + evt.data);
        const msg = JSON.parse(evt.data);
        const uri = location.pathname;
        const home = uri === '/';
        const archive = !home && uri.startsWith('/archive/');
        const tags = !home && !archive && uri.startsWith('/tags/');
        const single = !home && !archive && !tags && uri.startsWith('/' + msg.type + '/');
        const entryEl = document.getElementById(msg.id);
        const reload = home || archive || (single && !!entryEl);
        if (reload) {
            if (!home && !archive && msg.deleted) {
                location.href = '/';
            } else {
                reloadEntry(msg.type, msg.id, msg.deleted, single, entryEl);
            }
        }
    }
}

function reloadEntry(type, id, deleted, single, entryEl) {
    if (entryEl) {
        if (deleted) {
            entryEl.remove();
        } else {
            const xhr = new XMLHttpRequest();
            const typeIdPath = type + '/' + id;
            xhr.open('GET', '/' + typeIdPath + '.html', false);
            xhr.send();
            if (xhr.readyState === XMLHttpRequest.DONE) {
                if (xhr.status === 200) {
                    const lDoc = document.implementation.createHTMLDocument();
                    lDoc.documentElement.innerHTML = xhr.responseText;
                    const lMainEl = lDoc.getElementsByTagName('main')[0];
                    if (!single) {
                        const lHeaderEl = lMainEl.getElementsByTagName('header')[0];
                        lHeaderEl.innerHTML += '<span class="links"><a href="/' + typeIdPath + '.html" class="permalink"><i class="fa-solid fa-link"></i></a></span>';
                    }
                    entryEl.outerHTML = lMainEl.innerHTML;
                } else {
                    console.error('failed to reload content for: ' + typeIdPath);
                }
            }
        }
    } else {
        location.reload();
    }
}
