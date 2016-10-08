(function() {

  function saveJob() {
    var scheduling = document.getElementById('scheduling-prefs');
    scheduling = scheduling.getElementsByTagName('input');
    var jobJSON = {
      ID: document.getElementById('job-id').value,
      Name: document.getElementById('job-name').value,
      Tasks: window.encodeTasks(),
      MaxInstances: parseNumValue(scheduling[0], 'Max instances'),
      Priority: parseNumValue(scheduling[1], 'Priority'),
      NumCPU: parseNumValue(scheduling[2], 'CPUs'),
      MemUsage: parseNumValue(scheduling[3], 'Memory')
    };

    if (jobJSON.Priority > 0 && jobJSON.NumCPU === 0 &&
        jobJSON.MaxInstances === 0 && jobJSON.MemUsage === 0) {
      throw "job's scheduling is unbounded";
    }

    var form = document.createElement('form');
    var input = document.createElement('input');
    input.value = JSON.stringify(jobJSON);
    input.name = 'job';
    form.appendChild(input);
    form.method = 'POST';
    form.action = '/savejob';
    form.submit();
  }

  function parseNumValue(input, fieldName) {
    var num = parseInt(input.value);
    if (isNaN(num)) {
      throw 'bad ' + fieldName;
    }
    return num;
  }

  function deleteJob() {
    var id = document.getElementById('job-id').value;
    location = '/deletejob?id=' + id;
  }

  function registerCreators() {
    var tasks = document.getElementById('tasks');
    var keys = Object.keys(window.creators());
    for (var i = 0; i < keys.length; ++i) {
      (function(id) {
        var id = keys[i];
        var addButton = document.getElementById('add-' + id);
        addButton.onclick = function() {
          var el = window.creators()[id].create();
          tasks.appendChild(el);
        };
      })(keys[i]);
    }
  }

  window.addEventListener('load', function() {
    var saveButton = document.getElementsByClassName('save-button')[0];
    var deleteButton = document.getElementsByClassName('delete-button')[0];
    saveButton.addEventListener('click', function() {
      try {
        saveJob();
      } catch (e) {
        alert('Failed to save job: ' + e);
      }
    });
    deleteButton.addEventListener('click', deleteJob);
    registerCreators();
  });

})();
