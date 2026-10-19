package management

// This file declares the runtime of the shared components: the glyph lookup, the
// listbox, the chip grid and the label refresh that a language switch triggers.
// The glyph table is injected from pageIcons, so markup and runtime icons can
// never drift apart.

const templateScriptComponentsTemplate = `
var PAGE_ICONS = {{ICONS}};

// iconMarkup builds one glyph element. Callers pass literals only, so no page
// data ever reaches innerHTML.
function iconMarkup(name, className) {
  var glyph = PAGE_ICONS[name];
  if (!glyph) {
    return null;
  }
  var svg = document.createElementNS('http://www.w3.org/2000/svg', 'svg');
  svg.setAttribute('class', className ? 'icon ' + className : 'icon');
  svg.setAttribute('viewBox', '0 0 24 24');
  svg.setAttribute('fill', 'none');
  svg.setAttribute('stroke', 'currentColor');
  svg.setAttribute('stroke-width', '2');
  svg.setAttribute('stroke-linecap', 'round');
  svg.setAttribute('stroke-linejoin', 'round');
  svg.setAttribute('aria-hidden', 'true');
  svg.innerHTML = glyph;
  return svg;
}

function buildIconButton(className, iconName, label, handler) {
  var button = document.createElement('button');
  button.className = 'btn-icon' + (className ? ' ' + className : '');
  button.type = 'button';
  button.title = label;
  button.setAttribute('aria-label', label);
  button.appendChild(iconMarkup(iconName, ''));
  button.addEventListener('click', handler);
  return button;
}

// ---------------------------------------------------------------------------
// Listbox: the page's own select. A native <select> cannot be themed, so every
// choice on the page mounts this component instead. Options carry an i18n key
// so a language switch re-renders the labels.
// ---------------------------------------------------------------------------
function mountSelect(config) {
  var container = config && config.container;
  if (!container) {
    return null;
  }
  var options = config.options || [];
  var value = config.value === undefined ? (options[0] ? options[0].value : '') : config.value;
  var onChange = config.onChange;

  container.className = container.className ? container.className + ' select' : 'select';
  container.textContent = '';

  var trigger = document.createElement('button');
  trigger.type = 'button';
  trigger.className = 'select-trigger';
  trigger.setAttribute('role', 'combobox');
  trigger.setAttribute('aria-haspopup', 'listbox');
  trigger.setAttribute('aria-expanded', 'false');
  if (config.labelledBy) {
    trigger.setAttribute('aria-labelledby', config.labelledBy);
  }
  if (config.titleKey) {
    trigger.setAttribute('title', t(config.titleKey));
    trigger.setAttribute('aria-label', t(config.titleKey));
  }

  var label = document.createElement('span');
  label.className = 'select-label';
  trigger.appendChild(label);

  var arrow = iconMarkup('chevron', 'select-arrow');
  if (arrow) {
    trigger.appendChild(arrow);
  }

  var menu = document.createElement('div');
  menu.className = 'select-menu';
  menu.setAttribute('role', 'listbox');
  menu.hidden = true;

  container.appendChild(trigger);
  container.appendChild(menu);

  var activeIndex = -1;

  function optionLabel(option) {
    if (option.i18n) {
      return t(option.i18n);
    }
    return option.label === undefined ? String(option.value) : option.label;
  }

  function optionNodes() {
    return menu.querySelectorAll('.select-option');
  }

  function indexOfValue(target) {
    for (var index = 0; index < options.length; index++) {
      if (options[index].value === target) {
        return index;
      }
    }
    return -1;
  }

  function updateTriggerLabel() {
    var index = indexOfValue(value);
    label.textContent = index >= 0 ? optionLabel(options[index]) : '';
  }

  function renderMenu() {
    menu.textContent = '';
    for (var index = 0; index < options.length; index++) {
      var option = options[index];
      var node = document.createElement('div');
      node.className = 'select-option' + (option.value === value ? ' is-selected' : '');
      node.setAttribute('role', 'option');
      node.setAttribute('aria-selected', option.value === value ? 'true' : 'false');
      node.setAttribute('data-value', option.value);
      node.textContent = optionLabel(option);
      node.addEventListener('click', (function (selected) {
        return function (event) {
          event.stopPropagation();
          choose(selected);
        };
      })(option.value));
      menu.appendChild(node);
    }
    updateTriggerLabel();
    activeIndex = indexOfValue(value);
  }

  function setActive(index) {
    var nodes = optionNodes();
    if (!nodes.length) {
      return;
    }
    activeIndex = (index + nodes.length) % nodes.length;
    for (var position = 0; position < nodes.length; position++) {
      nodes[position].classList.toggle('is-active', position === activeIndex);
    }
    if (nodes[activeIndex].scrollIntoView) {
      nodes[activeIndex].scrollIntoView({ block: 'nearest' });
    }
  }

  function openMenu() {
    if (state.openSelect && state.openSelect !== component) {
      state.openSelect.close();
    }
    menu.hidden = false;
    container.classList.add('is-open');
    trigger.setAttribute('aria-expanded', 'true');
    // Flip the popup when the field sits too close to the bottom of the viewport,
    // which happens inside scrollable dialogs.
    menu.classList.remove('is-dropup');
    var rect = menu.getBoundingClientRect();
    if (rect.bottom > window.innerHeight - 8 && rect.top - rect.height > 8) {
      menu.classList.add('is-dropup');
    }
    setActive(indexOfValue(value));
    state.openSelect = component;
  }

  function closeMenu() {
    menu.hidden = true;
    container.classList.remove('is-open');
    trigger.setAttribute('aria-expanded', 'false');
    if (state.openSelect === component) {
      state.openSelect = null;
    }
  }

  function choose(next) {
    var changed = next !== value;
    value = next;
    renderMenu();
    closeMenu();
    trigger.focus();
    if (changed && onChange) {
      onChange(next);
    }
  }

  trigger.addEventListener('click', function (event) {
    event.stopPropagation();
    if (menu.hidden) {
      openMenu();
    } else {
      closeMenu();
    }
  });

  container.addEventListener('keydown', function (event) {
    var key = event.key;
    if (key === 'Escape') {
      if (menu.hidden) {
        return;
      }
      event.preventDefault();
      event.stopPropagation();
      closeMenu();
      trigger.focus();
      return;
    }
    if (key === 'Tab') {
      closeMenu();
      return;
    }
    if (key !== 'Enter' && key !== ' ' && key !== 'ArrowDown' && key !== 'ArrowUp' && key !== 'Home' && key !== 'End') {
      return;
    }
    event.preventDefault();
    if (menu.hidden) {
      openMenu();
      return;
    }
    if (key === 'ArrowDown') {
      setActive(activeIndex + 1);
    } else if (key === 'ArrowUp') {
      setActive(activeIndex - 1);
    } else if (key === 'Home') {
      setActive(0);
    } else if (key === 'End') {
      setActive(optionNodes().length - 1);
    } else {
      var nodes = optionNodes();
      if (nodes[activeIndex]) {
        choose(nodes[activeIndex].getAttribute('data-value'));
      }
    }
  });

  var component = {
    element: container,
    getValue: function () {
      return value;
    },
    setValue: function (next) {
      value = next;
      renderMenu();
    },
    setOptions: function (next) {
      options = next || [];
      renderMenu();
    },
    refresh: function () {
      renderMenu();
    },
    close: closeMenu
  };

  state.selects.push(component);
  renderMenu();
  return component;
}

function closeOpenSelect() {
  if (state.openSelect) {
    state.openSelect.close();
  }
}

// ---------------------------------------------------------------------------
// Chip grid: a multi-select of toggle chips for value sets such as the model
// allow-list.
// ---------------------------------------------------------------------------
function mountChipGrid(config) {
  var container = config && config.container;
  if (!container) {
    return null;
  }
  var items = config.items || [];
  var selected = {};
  var initial = config.selected || [];
  for (var index = 0; index < initial.length; index++) {
    if (initial[index]) {
      selected[String(initial[index]).toLowerCase()] = true;
    }
  }

  container.className = container.className ? container.className + ' chip-grid' : 'chip-grid';
  container.textContent = '';

  if (!items.length) {
    var empty = document.createElement('span');
    empty.className = 'hint';
    empty.textContent = '—';
    container.appendChild(empty);
  }

  var nodes = [];
  for (var position = 0; position < items.length; position++) {
    var item = items[position];
    var chip = document.createElement('button');
    chip.type = 'button';
    chip.className = 'chip';
    chip.setAttribute('data-value', item.value);
    var isSelected = selected[String(item.value).toLowerCase()] === true;
    chip.classList.toggle('is-selected', isSelected);
    chip.setAttribute('aria-pressed', isSelected ? 'true' : 'false');
    var check = document.createElement('span');
    check.className = 'chip-check';
    check.textContent = '✓';
    chip.appendChild(check);
    var text = document.createElement('span');
    text.textContent = item.label === undefined ? item.value : item.label;
    chip.appendChild(text);
    chip.addEventListener('click', (function (node) {
      return function () {
        var next = !node.classList.contains('is-selected');
        node.classList.toggle('is-selected', next);
        node.setAttribute('aria-pressed', next ? 'true' : 'false');
      };
    })(chip));
    container.appendChild(chip);
    nodes.push(chip);
  }

  return {
    getSelected: function () {
      var out = [];
      for (var index = 0; index < nodes.length; index++) {
        if (nodes[index].classList.contains('is-selected')) {
          out.push(nodes[index].getAttribute('data-value'));
        }
      }
      return out;
    },
    setSelected: function (values) {
      var wanted = {};
      for (var index = 0; index < (values || []).length; index++) {
        if (values[index]) {
          wanted[String(values[index]).toLowerCase()] = true;
        }
      }
      for (var position = 0; position < nodes.length; position++) {
        var next = wanted[String(nodes[position].getAttribute('data-value')).toLowerCase()] === true;
        nodes[position].classList.toggle('is-selected', next);
        nodes[position].setAttribute('aria-pressed', next ? 'true' : 'false');
      }
    }
  };
}

// refreshComponentLabels re-resolves the labels of every mounted component after
// a language switch.
function refreshComponentLabels() {
  for (var index = 0; index < state.selects.length; index++) {
    if (state.selects[index].refresh) {
      state.selects[index].refresh();
    }
  }
}
`
