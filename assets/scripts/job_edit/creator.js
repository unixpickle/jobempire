(function() {

  function TaskCreator(id) {
    this._base = document.getElementsByClassName('task-' + id)[0];
  }

  TaskCreator.prototype.create = function() {
    var el = this._base.cloneNode(true);
    setupCreated(el);
    return el;
  };

  var specialHandlers = {
    'gorun': function(el) {
      var deleteArg = el.getElementsByClassName('delete-button')[0];
      var addArg = el.getElementsByClassName('add-button')[0];
      addArg.onclick = function() {
        var field = document.createElement('div');
        field.className = 'text-field';
        var valContainer = document.createElement('div');
        valContainer.className = 'field-value';
        var val = document.createElement('input');
        valContainer.appendChild(val);
        field.appendChild(valContainer);

        var existing = el.getElementsByClassName('text-field');
        el.insertBefore(field, existing[existing.length - 1].nextElementSibling);
      };
      deleteArg.onclick = function() {
        var fields = el.getElementsByClassName('text-field');
        if (fields.length > 1) {
          var f = fields[fields.length - 1];
          f.parentNode.removeChild(f);
        }
      };
    }
  };

  function setupCreated(el) {
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
    var id = window.taskElementID(el);
    if (specialHandlers.hasOwnProperty(id)) {
      specialHandlers[id](el);
    }
  }

  function setupInitial() {
    var tasks = document.getElementById('tasks');
    tasks = tasks.getElementsByClassName('task');
    for (var i = 0, len = tasks.length; i < len; ++i) {
      var el = tasks[i];
      setupCreated(el);
    }
  }

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

  window.addEventListener('load', setupInitial);

})();
