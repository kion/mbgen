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
        const uri = location.pathname;
        const home = uri === '/' || uri === '/index.html';
        const archive = !home && uri === '/archive/';
        const tags = !home && !archive && uri === '/tags/';
        if (!archive && !tags) {
            const msg = JSON.parse(evt.data);
            const single = !home && !archive && !tags && (uri.startsWith('/post/') || uri.startsWith('/page/'));
            if (!single && msg.op === 'create') {
                location.reload();
            } else {
                const exactSingle = single && uri.startsWith('/' + msg.type + '/' + msg.id);
                const removed = msg.op === 'delete' || msg.op === 'rename';
                if (exactSingle && removed) {
                    location.href = '/';
                } else if (!single || (exactSingle && msg.op === 'update')) {
                    const ceEl = document.getElementById(msg.id);
                    reloadEntry(msg.type, msg.id, removed, exactSingle, ceEl);
                }
            }
        }
    }
}

function reloadEntry(type, id, removed, exactSingle, ceEl) {
    if (ceEl) {
        if (removed) {
            ceEl.remove();
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
                    if (!exactSingle) {
                        const lHeaderEl = lMainEl.getElementsByTagName('header')[0];
                        lHeaderEl.innerHTML += '<span class="links"><a href="/' + typeIdPath + '.html" class="permalink"><i class="fa-solid fa-link"></i></a></span>';
                    }
                    ceEl.outerHTML = lMainEl.innerHTML;
                } else {
                    console.error('failed to reload content for: ' + typeIdPath);
                }
            }
        }
    } else {
        location.reload();
    }
}
