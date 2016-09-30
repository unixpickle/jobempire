(function() {

  function saveJob() {
    var jobJSON = {
      ID: document.getElementById('job-id').value,
      Name: document.getElementById('job-name').value,
      Tasks: [],
      MaxInstances: 0,
      Priority: 0,
      NumCPU: 0,
    };

    // TODO: populate the rest of the fields from the UI.

    var form = document.createElement('form');
    var input = document.createElement('input');
    input.value = JSON.stringify(jobJSON);
    input.name = 'job';
    form.appendChild(input);
    form.method = 'POST';
    form.action = '/savejob';
    form.submit();
  }

  function deleteJob() {
    var id = document.getElementById('job-id').value;
    location = '/deletejob?id=' + id;
  }

  window.addEventListener('load', function() {
    var saveButton = document.getElementsByClassName('save-button')[0];
    var deleteButton = document.getElementsByClassName('delete-button')[0];
    saveButton.addEventListener('click', saveJob);
    deleteButton.addEventListener('click', deleteJob);
  });

})();
