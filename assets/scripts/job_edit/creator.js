(function() {

  function TaskCreator(id) {
    this._base = document.getElementsByClassName('task-' + id)[0];
  }

  TaskCreator.prototype.create = function() {
    var el = this._base.cloneNode(true);
    el.getElementsByClassName('task-delete')[0].onclick = function() {
      el.parentNode.removeChild(el);
    };
    el.getElementsByClassName('task-moveup')[0].onclick = function() {
      var last = el.previousElementSibling;
      if (last) {
        el.parentNode.insertBefore(el, last);
      }
    };
    el.getElementsByClassName('task-movedown')[0].onclick = function() {
      var next = el.nextElementSibling;
      if (next) {
        el.parentNode.insertBefore(next, el);
      }
    };
    return el;
  };

  var creatorIDs = ['filetransfer', 'gorun', 'exit'];
  var creators = null;
  window.creators = function() {
    if (creators === null) {
      creators = {};
      for (var i = 0; i < creatorIDs.length; ++i) {
        var id = creatorIDs[i];
        creators[id] = new TaskCreator(id);
      }
    }
    return creators;
  };

})();
