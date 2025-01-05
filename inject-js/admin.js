(function() {
    renderAdmin();
})();

function renderAdmin() {
    const uri = location.pathname;
    if (uri === '/' || uri === '/index.html') {
        const headerEl = document.getElementsByTagName('header')[0];
        renderAdminCreateButtons(headerEl);
    }
    const archive = uri.startsWith('/archive/');
    const tags = !archive && uri.startsWith('/tags/');
    if (!archive && !tags) {
        const mainEl = document.getElementsByTagName('main')[0];
        const contentEntryEls = mainEl.getElementsByClassName('content-entry');
        if (contentEntryEls.length) {
            for (let i = 0; i < contentEntryEls.length; i++) {
                const contentEntryEl = contentEntryEls[i];
                const entryType = contentEntryEl.getAttribute('data-type');
                const entryId = contentEntryEl.getAttribute('id');
                renderContentEntryAdminLinks(entryType, entryId, contentEntryEl);
            }
        }
    }
}

function renderAdminCreateButtons(headerEl) {
    const adminCreateHtml =
        '<section class="admin-create">' +
            '<button class="admin-btn" id="admin-create-page"><i class="fa-solid fa-square-plus"></i>Create New Page</button>' +
            '<button class="admin-btn" id="admin-create-post"><i class="fa-solid fa-calendar-plus"></i>Create New Post</button>' +
        '</section>';
    headerEl.outerHTML += adminCreateHtml;
    const adminCreatePageBtn = document.getElementById('admin-create-page');
    adminCreatePageBtn.onclick = function () {
        adminCreatePage();
    }
    const adminCreatePostBtn = document.getElementById('admin-create-post');
    adminCreatePostBtn.onclick = function () {
        adminCreatePost();
    }
}

function renderContentEntryAdminLinks(entryType, entryId, contentEntryEl) {
    const headerEl = contentEntryEl.getElementsByTagName('header')[0];
    const linksEl = headerEl.getElementsByClassName('links')[0];
    const adminEditHtml = '<a class="admin-link admin-edit"><i class="fa-solid fa-edit"></i></a>'
    const adminDeleteHtml = '<a class="admin-link admin-delete"><i class="fa-solid fa-trash-can"></i></a>'
    if (linksEl) {
        linksEl.innerHTML = linksEl.innerHTML + adminEditHtml + adminDeleteHtml;
    } else {
        headerEl.innerHTML += '<span class="links">' + adminEditHtml + adminDeleteHtml + '</span>';
    }
    registerAdminEditDeleteEventHandler(entryType, entryId, contentEntryEl);
}

function registerAdminEditDeleteEventHandler(entryType, entryId, contentEntryEl) {
    const adminEditEl = contentEntryEl.getElementsByClassName('admin-edit')[0];
    adminEditEl.onclick = function () {
        adminEdit(entryType, entryId, contentEntryEl);
    }
    const adminDeleteEl = contentEntryEl.getElementsByClassName('admin-delete')[0];
    adminDeleteEl.onclick = function () {
        adminDelete(entryType, entryId, contentEntryEl);
    }
}

function adminEdit(entryType, entryId, contentEntryEl) {
    const entryTypeId = entryType + '--' + entryId;
    const entryEditElId = 'admin-edit-' + entryTypeId;
    if (!document.getElementById(entryEditElId)) {
        const xhr = new XMLHttpRequest();
        xhr.open('GET', '/admin-edit?type=' + entryType + '&id=' + entryId, false);
        xhr.send();
        if (xhr.readyState === XMLHttpRequest.DONE) {
            if (xhr.status === 200) {
                const originalContent = contentEntryEl.outerHTML;
                const contentEl = contentEntryEl.getElementsByClassName('content')[0];
                contentEl.innerHTML =
                    '<div class="admin-edit">' +
                        '<textarea id="' + entryEditElId + '"></textarea>' +
                    '</div>';
                const contentEditor = new EasyMDE({
                    element: document.getElementById(entryEditElId),
                    toolbar: [
                        {
                            name: "save",
                            action: () => {
                                const xhr = new XMLHttpRequest();
                                xhr.open('POST', '/admin-edit?type=' + entryType + '&id=' + entryId, false);
                                xhr.setRequestHeader('Content-Type', 'text/markdown');
                                xhr.send(contentEditor.value());
                                if (xhr.readyState === XMLHttpRequest.DONE) {
                                    if (xhr.status === 200) {
                                        contentEntryEl.outerHTML = xhr.responseText;
                                        contentEntryEl = document.getElementById(entryId);
                                        if (location.pathname !== '/' + entryType + '/' + entryId + '.html') {
                                            const ceHeaderEl = contentEntryEl.getElementsByTagName('header')[0];
                                            ceHeaderEl.innerHTML += '<span class="links"><a href="/' + entryType + '/' + entryId + '.html" class="permalink"><i class="fa-solid fa-link"></i></a></span>';
                                        }
                                        renderContentEntryAdminLinks(entryType, entryId, contentEntryEl);
                                    } else {
                                        console.error('Failed to save content for ' + entryType + '/' + entryId + ': ' + xhr.responseText);
                                    }
                                }
                            },
                            className: "fa fa-save",
                        },
                        {
                            name: "cancel",
                            action: () => {
                                contentEntryEl.outerHTML = originalContent;
                                contentEntryEl = document.getElementById(entryId);
                                registerAdminEditDeleteEventHandler(entryType, entryId, contentEntryEl);
                            },
                            className: "fa fa-circle-xmark",
                            attributes: {
                                "style": "float:right"
                            },
                        },
                    ],
                    blockStyles: {
                        bold: "**",
                        italic: "_",
                        code: "```",
                    },
                    indentWithTabs: false,
                    status: false,
                    spellChecker: false,
                    autoDownloadFontAwesome: false,
                    autofocus: true,
                    initialValue: xhr.responseText.replace(/^(---\n.*)\n---/gm, '$1\n\n---'),
                });
                const doc = contentEditor.codemirror.getDoc();
                const text = doc.getValue();
                const firstDelimiter = text.indexOf('---');
                const secondDelimiter = text.indexOf('---', firstDelimiter + 3);
                if (secondDelimiter !== -1) {
                    const line = doc.posFromIndex(secondDelimiter).line + 2;
                    doc.setCursor(line, 0);
                }
            } else {
                console.error('Failed to load content for ' + entryType + '/' + entryId + ': ' + xhr.responseText);
            }
        }
    }
}

function adminDelete(entryType, entryId, contentEntryEl) {
    if (confirm('Are you sure you want to delete ' + entryType + '/' + entryId + '?')) {
        const xhr = new XMLHttpRequest();
        xhr.open('POST', '/admin-delete?type=' + entryType + '&id=' + entryId, false);
        xhr.send();
        if (xhr.readyState === XMLHttpRequest.DONE) {
            if (xhr.status === 204) {
                contentEntryEl.remove();
                if (location.pathname === '/' + entryType + '/' + entryId + '.html') {
                    location.href = '/';
                }
            } else {
                console.error('Failed to delete ' + entryType + '/' + entryId + ': ' + xhr.responseText);
            }
        }
    }
}

function adminCreatePage(pId) {
    const pageId = prompt('Page ID:', pId);
    if (pageId) {
        const xhr = new XMLHttpRequest();
        xhr.open('POST', '/admin-create?type=page&id=' + pageId, false);
        xhr.send();
        if (xhr.readyState === XMLHttpRequest.DONE) {
            if (xhr.status === 201) {
                location.href = xhr.getResponseHeader('Location');
            } else {
                alert('Failed to create page: ' + xhr.responseText);
                adminCreatePage(pageId);
            }
        }
    }
}

function adminCreatePost(pId) {
    const postId = prompt('Post ID:', pId);
    if (postId) {
        const xhr = new XMLHttpRequest();
        xhr.open('POST', '/admin-create?type=post&id=' + postId, false);
        xhr.send();
        if (xhr.readyState === XMLHttpRequest.DONE) {
            if (xhr.status === 201) {
                location.href = xhr.getResponseHeader('Location');
            } else {
                alert('Failed to create post: ' + xhr.responseText);
                adminCreatePost(postId);
            }
        }
    }
}
