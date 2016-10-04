(function() {

  var toggleButton;
  var TOGGLE_CLASS = 'hide-done-slaves';

  function toggleShowing() {
    var classes = document.body.className.split(' ');
    var idx = classes.indexOf(TOGGLE_CLASS);
    if (idx >= 0) {
      classes.splice(idx, 1);
      toggleButton.textContent = 'Hide Done';
    } else {
      classes.push(TOGGLE_CLASS);
      toggleButton.textContent = 'Show All';
    }
    document.body.className = classes.join(' ');
  }

  window.addEventListener('load', function() {
    toggleButton = document.getElementById('toggle-showall');
    toggleButton.addEventListener('click', toggleShowing);
  });

})();
