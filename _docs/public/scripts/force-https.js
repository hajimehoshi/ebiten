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
    window.location.href = url;
})();
