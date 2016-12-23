var http = require('http');

module.exports = function (context, callback) {
  var body = context.request.body;
  if (body && body.constructor === Array) {
    var counter = 0;
    var result = {
      responses: []
    };
    body.forEach(function (item, idx) {
      var req = http.request(
              {host: 'blogcount', method: 'POST'},
              function (response) {
                var str = '';
                response.on('data', function (chunk) {
                  str += chunk;
                });
                response.on('end', function () {
                  responses[idx] = str;
                });
              });
      req.write(JSON.stringify(item));
      req.end();
      counter++;
    });
    result.count = counter;
    callback(200, JSON.stringify(results));
  } else {
    callback(400, "No array is passed in. Was given: " + JSON.stringify(body));
  }
};
