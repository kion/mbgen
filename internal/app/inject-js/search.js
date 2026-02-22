let searchOutputEl;
let searchSummaryEl;
let searchResultsEl;
let searchPagerEl;
let pageSize;
let searchResults;
let searchIndex;

const searchHelpToggleLabelDisabled = '(?)';
const searchHelpToggleLabelEnabled = 'Hide Help';
const searchHelpContent =
'<ul>' +
    '<li>search is <i>case-insensitive</i> and <i>unicode-friendly</i></li>' +
    '<li>returns records matching <i>any</i> / <i>at least one of</i> the search terms</li>' +
    '<li>individual search terms are separated by spaces</li>' +
    '<li>individual search term is one of the following:' +
        '<ul>' +
            '<li>a single word</li>' +
            '<li>multiple words enclosed in double quotes</li>' +
        '</ul>' +
    '</li>' +
    '<li><pre>term1+term2</pre> = find records matching <b>both</b> <pre>term1</pre> <b>and</b> <pre>term2</pre></li>' +
    '<li><pre>term1-term2</pre> = find records matching <pre>term1</pre>, but <b>not</b> matching <pre>term2</pre></li>' +
    '<li>combine search terms/operators as needed, for example:' +
        '<ul>' +
            '<li><pre>food restaurant</pre> - records matching <b>either</b> <pre>food</pre> <b>or</b> <pre>restaurant</pre></li>' +
            '<li><pre>"terraforming mars"</pre> - records matching <pre>terraforming mars</pre> <b>exactly</b> (i.e. both words in the exact given order)</li>' +
            '<li><pre>cooking+"tom yum"</pre> - records matching <b>both</b> <pre>cooking</pre> <b>and</b> <pre>tom yum</pre></li>' +
            '<li><pre>cycling-zwift</pre> - records matching <pre>cycling</pre>, but <b>not</b> matching <pre>zwift</pre></li>' +
            '<li><pre>"bike ride" strava+segment</pre> - records with <b>either</b> (1) an <b>exact</b> <pre>bike ride</pre> match or (2) matching <b>both</b> <pre>strava</pre> <b>and</b> <pre>segment</pre></li>' +
        '</ul>' +
    '</li>' +
'</ul>';

function search(query) {
    if (query) {
        if (!searchIndex) {
            const xhr = new XMLHttpRequest();
            xhr.open('GET', '/search.json', false);
            xhr.send();
            if (xhr.readyState === XMLHttpRequest.DONE) {
                if (xhr.status === 200) {
                    searchIndex = JSON.parse(xhr.responseText);
                    processQuery(query, searchIndex);
                } else {
                    console.error("Failed to load search index");
                }
            }
        } else {
            processQuery(query, searchIndex);
        }
    }
}

function processQuery(query, searchIndex) {
    // ================================================================================
    // clear previous search results
    // ================================================================================
    searchSummaryEl.innerHTML = '';
    searchResultsEl.innerHTML = '';
    searchPagerEl.style.display = 'none';
    // ================================================================================
    // parse query
    // ================================================================================
    let terms = {};
    let term = '';
    let phrase = false;
    for (let i = 0; i < query.length; i++) {
        let c = query.charAt(i);
        if (c === '"') {
            phrase = !phrase;
        } else if (c === ' ' && !phrase) {
            if (term.length > 0) {
                terms[term] = [];
                term = '';
            }
        } else {
            term += c;
        }
    }
    if (term.length > 0) {
        terms[term] = [];
    }
    for (let term in terms) {
        let subTerms = [];
        let subTerm = '';
        let exclude = false;
        for (let j = 0; j < term.length; j++) {
            let c = term.charAt(j);
            if (c === '+' || c === '-') {
                if (subTerm.length > 0) {
                    subTerms.push({term: subTerm, exclude: exclude});
                    subTerm = '';
                }
                exclude = c === '-';
            } else {
                subTerm += c;
            }
        }
        if (subTerm.length > 0) {
            subTerms.push({term: subTerm, exclude: exclude});
        }
        terms[term] = subTerms;
    }
    // ================================================================================
    // perform search
    // ================================================================================
    searchResults = [];
    let foundCnt = 0;
    for (let typeId in searchIndex) {
        let content = searchIndex[typeId];
        let termMatch = false;
        for (let term in terms) {
            let subTerms = terms[term];
            let subTermsMatch = true;
            for (let j = 0; j < subTerms.length; j++) {
                let subTerm = subTerms[j];
                let subTermInc = content.includes(subTerm.term.toLowerCase());
                if (subTerm.exclude) {
                    if (subTermInc) {
                        subTermsMatch = false;
                        break;
                    }
                } else {
                    if (!subTermInc) {
                        subTermsMatch = false;
                        break;
                    }
                }
            }
            if (subTermsMatch) {
                termMatch = true;
                break;
            }
        }
        if (termMatch) {
            foundCnt++;
            searchResults.push(typeId);
        }
    }
    // ================================================================================
    // render results
    // ================================================================================
    if (foundCnt > 0) {
        searchSummaryEl.innerHTML = '<label>' + foundCnt + ' record' + (foundCnt > 1 ? 's' : '') + ' found</label>';
        renderNextSearchResultsPage();
    } else {
        searchSummaryEl.innerHTML = '<label>No records found</label>';
    }
    searchSummaryEl.style.display = 'block';
    searchOutputEl.style.display = 'block';
    // ================================================================================
}

function renderNextSearchResultsPage() {
    for (let i = 0; i < pageSize; i++) {
        if (searchResults.length > 0) {
            const typeId = searchResults.shift();
            processContent(typeId, searchResultsEl);
        }
    }
    if (searchResults.length > 0) {
        searchPagerEl.style.display = 'block';
    } else {
        searchPagerEl.style.display = 'none';
    }
    if (typeof renderAdmin === 'function') {
        renderAdmin();
    }
}

function processContent(typeId) {
    const xhr = new XMLHttpRequest();
    xhr.open('GET', '/' + typeId + '.html', false);
    xhr.send();
    if (xhr.readyState === XMLHttpRequest.DONE) {
        if (xhr.status === 200) {
            const lDoc = document.implementation.createHTMLDocument();
            lDoc.documentElement.innerHTML = xhr.responseText;
            const lMainEl = lDoc.getElementsByTagName('main')[0];
            const lHeaderEl = lMainEl.getElementsByTagName('header')[0];
            lHeaderEl.innerHTML += '<span class="links"><a href="/' + typeId + '.html" class="permalink"><i class="fa-solid fa-link"></i></a></span>';
            searchResultsEl.innerHTML += lMainEl.innerHTML;
        } else {
            console.error("Failed to load content for: " + typeId);
        }
    }
}

window.onload = function() {
    searchOutputEl = document.getElementById('search-output');
    searchOutputEl.style.display = 'none';
    searchSummaryEl = document.getElementById('search-summary');
    searchResultsEl = document.getElementById('search-results');
    searchPagerEl = document.getElementById('search-pager');
    pageSize = parseInt(searchPagerEl.getAttribute('data-page-size'));
    let searchHelpContentEl = document.getElementById('search-help-content');
    searchHelpContentEl.style.display = 'none';
    searchHelpContentEl.innerHTML = searchHelpContent;
    const searchQueryEl = document.getElementById('search-query');
    const urlParams = new URLSearchParams(window.location.search);
    const query = urlParams.get('q');
    if (query) {
        searchQueryEl.value = query;
        search(query);
    }
    searchQueryEl.addEventListener('keydown', function(event) {
        if (event.key === 'Enter') {
            event.preventDefault();
            const query = searchQueryEl.value;
            if (query) {
                window.history.pushState({}, '', '/search.html?q=' + encodeURIComponent(query));
                search(query);
            }
        }
    });
    const searchSubmitEl = document.getElementById('search-submit');
    searchSubmitEl.innerHTML = 'Search';
    searchSubmitEl.addEventListener('click', function(event) {
        event.preventDefault();
        const query = searchQueryEl.value;
        if (query) {
            window.history.pushState({}, '', '/search.html?q=' + encodeURIComponent(query));
            search(query);
        }
    });
    const searchLoadNextPageEl = document.getElementById('search-load-next-page');
    searchLoadNextPageEl.addEventListener('click', function(event) {
        event.preventDefault();
        renderNextSearchResultsPage();
    });
    const searchHelpToggleEl = document.getElementById('search-help-toggle');
    searchHelpToggleEl.innerHTML = searchHelpToggleLabelDisabled;
    searchHelpToggleEl.addEventListener('click', function(event) {
        event.preventDefault();
        if (searchHelpContentEl.style.display === 'none') {
            searchHelpContentEl.style.display = 'block';
            searchHelpToggleEl.innerHTML = searchHelpToggleLabelEnabled;
        } else {
            searchHelpContentEl.style.display = 'none';
            searchHelpToggleEl.innerHTML = searchHelpToggleLabelDisabled;
        }
    });
}
