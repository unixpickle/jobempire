(function() {

  function encodeTasks() {
    var taskContainer = document.getElementById('tasks');
    var tasks = taskContainer.getElementsByClassName('task');
    var res = [];
    for (var i = 0, len = tasks.length; i < len; ++i) {
      res.push(encodeTask(tasks[i]));
    }
    return res;
  }

  function encodeTask(el) {
    var id = taskElementID(el);
    return {
      filetransfer: encodeFileTransfer,
      gorun: encodeGoRun,
      exit: encodeExit
    }[id](el);
  }

  function encodeFileTransfer(el) {
    var inputs = el.getElementsByTagName('input');
    return {
      FileTransfer: {
        ToSlave: !!inputs[0].checked,
        MasterPath: inputs[1].value,
        SlavePath: inputs[2].value
      },
    };
  }

  function encodeGoRun(el) {
    var inputs = el.getElementsByTagName('input');
    var res = {
      GoRun: {
        GoSourceDir: inputs[0].value,
        Arguments: []
      }
    };
    for (var i = 1, len = inputs.length; i < len; ++i) {
      res.GoRun.Arguments.push(inputs[i].value);
    }
    return res;
  }

  function encodeExit(el) {
    return {'Exit': {}};
  }

  function taskElementID(el) {
    var classes = el.className.split(' ');
    for (var i = 0, len = classes.length; i < len; ++i) {
      if (/task-/.exec(classes[i])) {
        return classes[i].substr(5);
      }
    }
    throw 'element does not have task ID';
  }

  window.encodeTasks = encodeTasks;
  window.taskElementID = taskElementID;

})();
