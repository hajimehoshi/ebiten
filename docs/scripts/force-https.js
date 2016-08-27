'use strict';

(function() {
    var location = window.location;
    if (location.hostname === 'localhost' || location.hostname === '127.0.0.1') {
        return;
    }
    var protocol = location.protocol;
    if (protocol === 'https:') {
        return;
    }
    var url = 'https:' + window.location.href.substring(protocol.length);
    // Pass the referrer info to the new URL for Google Analytics.
    if (document.referrer) {
        var referrerPart = 'referrer=' + encodeURIComponent(document.referrer);
        url += (location.search ? '&' : '?') + referrerPart;
    }
    window.location.href = url;
})();
