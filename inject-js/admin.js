const supportedMediaFileExt = ['.jpg', '.jpeg', '.png', '.gif', '.mp4', '.mkv', '.mov'];
const supportedMediaFileExtStr = supportedMediaFileExt.join(',');

(function() {
    renderAdmin();
})();

function renderAdmin() {
    const uri = location.pathname;
    const home = uri === '/' || uri === '/index.html';
    if (home) {
        const headerEl = document.getElementsByTagName('header')[0];
        renderAdminCreateButtons(headerEl);
    }
    const archive = !home && uri === '/archive/';
    const tags = !home && !archive && uri === '/tags/';
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
            '<section class="admin-controls">' +
                '<button class="admin-btn" id="admin-create-page"><i class="fa-solid fa-square-plus"></i>Create New Page</button>' +
                '<button class="admin-btn" id="admin-create-post"><i class="fa-solid fa-calendar-plus"></i>Create New Post</button>' +
            '</section>' +
        '</section>';
    headerEl.outerHTML += adminCreateHtml;
    const adminCreatePageBtn = document.getElementById('admin-create-page');
    adminCreatePageBtn.onclick = function() {
        adminCreatePage();
    }
    const adminCreatePostBtn = document.getElementById('admin-create-post');
    adminCreatePostBtn.onclick = function() {
        adminCreatePost();
    }
}

function renderContentEntryAdminLinks(entryType, entryId, contentEntryEl) {
    const headerEl = contentEntryEl.getElementsByTagName('header')[0];
    const linksEl = headerEl.getElementsByClassName('links')[0];
    if (!linksEl || linksEl.getElementsByClassName('admin-link').length === 0) {
        // render admin links only if there are no admin links already
        const adminEditHtml = '<a class="admin-link admin-edit"><i class="fa-solid fa-edit"></i></a>'
        const adminMediaHtml = '<a class="admin-link admin-media"><i class="fa-solid fa-images"></i></a>'
        const adminDeleteHtml = '<a class="admin-link admin-delete"><i class="fa-solid fa-trash-can"></i></a>'
        if (linksEl) {
            linksEl.innerHTML = linksEl.innerHTML + adminEditHtml + adminMediaHtml + adminDeleteHtml;
        } else {
            headerEl.innerHTML += '<span class="links">' + adminEditHtml + adminMediaHtml + adminDeleteHtml + '</span>';
        }
    }
    registerAdminEventHandlers(entryType, entryId, contentEntryEl);
}

function registerAdminEventHandlers(entryType, entryId, contentEntryEl) {
    showAdminControls(contentEntryEl);
    const adminEditEl = contentEntryEl.getElementsByClassName('admin-edit')[0];
    adminEditEl.onclick = function() {
        hideAdminControls(contentEntryEl);
        adminEdit(entryType, entryId, contentEntryEl);
    }
    const adminDeleteEl = contentEntryEl.getElementsByClassName('admin-delete')[0];
    adminDeleteEl.onclick = function() {
        adminDelete(entryType, entryId, contentEntryEl);
    }
    const adminMediaEl = contentEntryEl.getElementsByClassName('admin-media')[0];
    adminMediaEl.onclick = function() {
        hideAdminControls(contentEntryEl);
        adminMedia(entryType, entryId, contentEntryEl);
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
                alert('failed to create page: ' + xhr.responseText);
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
                alert('failed to create post: ' + xhr.responseText);
                adminCreatePost(postId);
            }
        }
    }
}

function adminEdit(entryType, entryId, contentEntryEl) {
    const typeIdPath = entryType + '/' + entryId;
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
                    '<section class="admin-edit">' +
                        '<textarea id="' + entryEditElId + '"></textarea>' +
                        '<section class="admin-controls">' +
                            '<button class="admin-btn" id="' + entryEditElId + '-close"><i class="fa-solid fa-circle-xmark"></i>Close / Discard Changes</button>' +
                            '<button class="admin-btn" id="' + entryEditElId + '-save"><i class="fa-solid fa-save"></i>Save Changes</button>' +
                        '</section>' +
                    '</section>';
                const contentEditor = new EasyMDE({
                    element: document.getElementById(entryEditElId),
                    blockStyles: {
                        bold: "**",
                        italic: "_",
                        code: "```",
                    },
                    toolbar: false,
                    indentWithTabs: false,
                    status: false,
                    spellChecker: false,
                    autoDownloadFontAwesome: false,
                    autofocus: true,
                    initialValue: xhr.responseText.replace(/^(---\n.*)\n---/gm, '$1\n\n---'),
                });
                contentEditor.codemirror.setCursor(contentEditor.codemirror.lineCount(), 0);
                const entryEditSaveEl = document.getElementById(entryEditElId + '-save');
                entryEditSaveEl.onclick = function() {
                    const xhr = new XMLHttpRequest();
                    xhr.open('POST', '/admin-edit?type=' + entryType + '&id=' + entryId, false);
                    xhr.setRequestHeader('Content-Type', 'text/markdown');
                    xhr.send(contentEditor.value());
                    if (xhr.readyState === XMLHttpRequest.DONE) {
                        if (xhr.status === 200) {
                            contentEntryEl.outerHTML = xhr.responseText;
                            contentEntryEl = document.getElementById(entryId);
                            if (location.pathname !== '/' + typeIdPath + '.html') {
                                const ceHeaderEl = contentEntryEl.getElementsByTagName('header')[0];
                                ceHeaderEl.innerHTML += '<span class="links"><a href="/' + typeIdPath + '.html" class="permalink"><i class="fa-solid fa-link"></i></a></span>';
                            }
                            renderContentEntryAdminLinks(entryType, entryId, contentEntryEl);
                        } else {
                            alert('failed to save content');
                            console.error('failed to save content for ' + typeIdPath + ': ' + xhr.responseText);
                        }
                    }
                }
                const entryEditCloseEl = document.getElementById(entryEditElId + '-close');
                entryEditCloseEl.onclick = function() {
                    contentEntryEl.outerHTML = originalContent;
                    contentEntryEl = document.getElementById(entryId);
                    registerAdminEventHandlers(entryType, entryId, contentEntryEl);
                }
            } else {
                alert('failed to load content');
                console.error('failed to load content for ' + typeIdPath + ': ' + xhr.responseText);
            }
        }
    }
}

function adminDelete(entryType, entryId, contentEntryEl) {
    const typeIdPath = entryType + '/' + entryId;
    if (confirm('Are you sure you want to delete ' + typeIdPath + '?')) {
        const xhr = new XMLHttpRequest();
        xhr.open('POST', '/admin-delete?type=' + entryType + '&id=' + entryId, false);
        xhr.send();
        if (xhr.readyState === XMLHttpRequest.DONE) {
            if (xhr.status === 204) {
                if (location.pathname === '/' + typeIdPath + '.html') {
                    location.href = '/';
                } else {
                    contentEntryEl.remove();
                }
            } else {
                alert('failed to delete');
                console.error('failed to delete ' + typeIdPath + ': ' + xhr.responseText);
            }
        }
    }
}

function adminMedia(entryType, entryId, contentEntryEl) {
    const typeIdPath = entryType + '/' + entryId;
    const entryTypeId = entryType + '--' + entryId;
    const entryMediaElId = 'admin-media-' + entryTypeId;
    if (!document.getElementById(entryMediaElId)) {
        const xhr = new XMLHttpRequest();
        xhr.open('GET', '/admin-media?type='+ entryType + '&id=' + entryId, false);
        xhr.send();
        if (xhr.readyState === XMLHttpRequest.DONE) {
            if (xhr.status === 200) {
                const contentEl = contentEntryEl.getElementsByClassName('content')[0];
                contentEl.innerHTML =
                    '<section class="admin-media">' +
                        '<section id="' + entryMediaElId + '"></section>' +
                        '<form enctype="multipart/form-data" id="' + entryMediaElId + '-upload-form">' +
                            '<input type="file" multiple accept="' + supportedMediaFileExtStr + '" name="admin-media-upload-files" id="' + entryMediaElId + '-upload-file" style="display:none">' +
                        '</form>' +
                        '<section class="admin-controls">' +
                            '<button id="' + entryMediaElId + '-add" class="admin-btn"><i class="fa-solid fa-folder-plus"></i>Add Media</button>' +
                            '<button id="' + entryMediaElId + '-close" class="admin-btn"><i class="fa-solid fa-circle-xmark"></i>Close</button>' +
                        '</section>' +
                    '</section>';
                let mediaEditorEl = document.getElementById(entryMediaElId);
                mediaEditorEl.outerHTML = xhr.responseText;
                const handleMediaFormData = function(entryType, entryId, mediaUploadFormData) {
                    uploadMediaFormData(entryType, entryId, mediaUploadFormData,
                        function(responseText){
                            mediaEditorEl = contentEl.getElementsByClassName('admin-media')[0];
                            mediaEditorEl.outerHTML = responseText;
                            adminMedia(entryType, entryId, contentEntryEl);
                        },
                        function(responseText){
                            alert('failed to upload media');
                            console.error('failed to upload media for ' + typeIdPath + ': ' + responseText);
                        }
                    );
                }
                const entryMediaUploadFile = document.getElementById(entryMediaElId + '-upload-file');
                entryMediaUploadFile.onchange = function() {
                    const mediaUploadForm = document.getElementById(entryMediaElId + '-upload-form');
                    const mediaUploadFormData = new FormData(mediaUploadForm);
                    handleMediaFormData(entryType, entryId, mediaUploadFormData);
                }
                const entryMediaAddEl = document.getElementById(entryMediaElId + '-add');
                entryMediaAddEl.onclick = function() {
                    entryMediaUploadFile.click();
                }
                entryMediaAddEl.ondragover = function(ev) {
                    ev.preventDefault();
                }
                entryMediaAddEl.ondrop = function(ev) {
                    ev.preventDefault();
                    const dt = ev.dataTransfer;
                    const files = dt.files;
                    for (let i = 0; i < files.length; i++) {
                        const file = files[i];
                        if (supportedMediaFileExt.includes(file.name.substring(file.name.lastIndexOf('.')))) {
                            const mediaUploadForm = document.getElementById(entryMediaElId + '-upload-form');
                            const mediaUploadFormData = new FormData(mediaUploadForm);
                            mediaUploadFormData.append('admin-media-upload-files', file);
                            handleMediaFormData(entryType, entryId, mediaUploadFormData);
                        }
                    }
                }
                mediaEditorEl = contentEl.getElementsByClassName('admin-media')[0];
                const imageEls = mediaEditorEl.getElementsByClassName('image');
                const videoEls = mediaEditorEl.getElementsByClassName('video');
                const mediaEls = [...imageEls, ...videoEls];
                for (let i = 0; i < mediaEls.length; i++) {
                    const mediaEl = mediaEls[i];
                    const fileName = mediaEl.classList.contains('video')
                        ? mediaEl.getElementsByTagName('video')[0].getAttribute('src').split('/').pop()
                        : mediaEl.getElementsByTagName('a')[0].getAttribute('href').split('/').pop();
                    mediaEl.innerHTML +=
                        '<span class="admin-media-controls">' +
                            '<span class="admin-media-file-name">' + fileName + '</span>' +
                            '<span class="admin-media-delete" data-fn="' + fileName + '">' +
                                '<i class="fa-solid fa-trash-can"></i>' +
                            '</span>' +
                        '</span>';
                    const mediaDeleteEl = mediaEl.getElementsByClassName('admin-media-delete')[0];
                    mediaDeleteEl.onclick = function() {
                        if (confirm('Are you sure you want to delete ' + entryId + '/' + fileName + '?')) {
                            const xhr = new XMLHttpRequest();
                            xhr.open('DELETE', '/admin-media?type='+ entryType + '&id=' + entryId + '&fileName=' + fileName, false);
                            xhr.send();
                            if (xhr.readyState === XMLHttpRequest.DONE) {
                                if (xhr.status === 205) {
                                    mediaEditorEl.outerHTML = xhr.responseText;
                                    adminMedia(entryType, entryId, contentEntryEl);
                                } else {
                                    alert('failed to delete media');
                                    console.error('failed to delete media for ' + typeIdPath + ': ' + xhr.responseText);
                                }
                            }
                        }
                    }
                }
                const entryMediaCloseEl = document.getElementById(entryMediaElId + '-close');
                entryMediaCloseEl.onclick = function() {
                    const xhr = new XMLHttpRequest();
                    xhr.open('GET', '/' + typeIdPath + '.html', false);
                    xhr.send();
                    if (xhr.readyState === XMLHttpRequest.DONE) {
                        if (xhr.status === 200) {
                            const lDoc = document.implementation.createHTMLDocument();
                            lDoc.documentElement.innerHTML = xhr.responseText;
                            const lMainEl = lDoc.getElementsByTagName('main')[0];
                            const lContentEl = lMainEl.getElementsByClassName('content')[0];
                            const contentEl = contentEntryEl.getElementsByClassName('content')[0];
                            contentEl.outerHTML = lContentEl.outerHTML;
                            contentEntryEl = document.getElementById(entryId);
                            registerAdminEventHandlers(entryType, entryId, contentEntryEl);
                        } else {
                            console.error('failed to reload content for: ' + typeIdPath);
                        }
                    }
                }
            } else {
                alert('failed to load media');
                console.error('failed to load media for ' + typeIdPath + ': ' + xhr.responseText);
            }
        }
    }
}

function uploadMediaFormData(entryType, entryId, mediaUploadFormData, successCallbackFn, failureCallbackFn) {
    const xhr = new XMLHttpRequest();
    xhr.open('POST', '/admin-media?type='+ entryType + '&id=' + entryId, false);
    xhr.send(mediaUploadFormData);
    if (xhr.readyState === XMLHttpRequest.DONE) {
        if (xhr.status === 201) {
            successCallbackFn(xhr.responseText);
        } else {
            failureCallbackFn(xhr.responseText);
        }
    }
}

function hideAdminControls(contentEntryEl) {
    const adminLinkEls = contentEntryEl.getElementsByClassName('admin-link');
    for (let i = 0; i < adminLinkEls.length; i++) {
        adminLinkEls[i].style.display = 'none';
    }
}

function showAdminControls(contentEntryEl) {
    const adminLinkEls = contentEntryEl.getElementsByClassName('admin-link');
    for (let i = 0; i < adminLinkEls.length; i++) {
        adminLinkEls[i].style.display = 'inline-block';
    }
}
