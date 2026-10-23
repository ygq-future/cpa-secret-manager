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
  var isSearchable = Boolean(config.searchable);
  var searchQuery = '';

  container.className = container.className ? container.className + ' select' : 'select';
  if (isSearchable) {
    container.classList.add('is-searchable');
  }
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
  menu.className = 'select-menu' + (isSearchable ? ' has-search' : '');
  menu.setAttribute('role', 'listbox');
  menu.hidden = true;

  var searchInput = null;
  var clearBtn = null;
  var searchWrap = null;
  var optionsList = document.createElement('div');
  optionsList.className = 'select-options-list';

  if (isSearchable) {
    searchWrap = document.createElement('div');
    searchWrap.className = 'select-search-wrap';

    var searchIcon = iconMarkup('search', 'icon');
    if (searchIcon) {
      searchWrap.appendChild(searchIcon);
    }

    searchInput = document.createElement('input');
    searchInput.type = 'text';
    searchInput.className = 'select-search-input';
    searchInput.placeholder = config.searchPlaceholder || t('key_form.quota_models_search');
    searchInput.autocomplete = 'off';
    searchInput.spellcheck = false;
    searchInput.setAttribute('data-bwignore', 'true');
    searchInput.setAttribute('data-lpignore', 'true');
    searchInput.setAttribute('data-1p-ignore', 'true');
    searchWrap.appendChild(searchInput);

    clearBtn = document.createElement('button');
    clearBtn.type = 'button';
    clearBtn.className = 'select-clear-btn';
    var clearIcon = iconMarkup('close', 'icon');
    if (clearIcon) {
      clearBtn.appendChild(clearIcon);
    }
    clearBtn.title = t('keys.filter_clear');
    clearBtn.hidden = true;
    searchWrap.appendChild(clearBtn);

    menu.appendChild(searchWrap);
  }
  menu.appendChild(optionsList);

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
    return optionsList.querySelectorAll('.select-option');
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

  function filteredOptions() {
    if (!isSearchable || !searchQuery) {
      return options;
    }
    var query = searchQuery.trim().toLowerCase();
    if (!query) {
      return options;
    }
    return options.filter(function (opt) {
      var lbl = optionLabel(opt);
      var val = String(opt.value || '');
      return lbl.toLowerCase().indexOf(query) >= 0 || val.toLowerCase().indexOf(query) >= 0;
    });
  }

  function renderMenu() {
    optionsList.textContent = '';
    var list = filteredOptions();
    if (list.length === 0) {
      var empty = document.createElement('div');
      empty.className = 'select-empty';
      empty.textContent = t('key_form.quota_models_empty');
      optionsList.appendChild(empty);
      activeIndex = -1;
    } else {
      for (var index = 0; index < list.length; index++) {
        var option = list[index];
        var node = document.createElement('div');
        node.className = 'select-option' + (option.value === value ? ' is-selected' : '');
        node.setAttribute('role', 'option');
        node.setAttribute('aria-selected', option.value === value ? 'true' : 'false');
        node.setAttribute('data-value', option.value);
        var labelSpan = document.createElement('span');
        labelSpan.className = 'select-option-label';
        labelSpan.textContent = optionLabel(option);
        node.appendChild(labelSpan);
        if (option.configured) {
          var checkIcon = iconMarkup('check', 'select-option-check');
          if (checkIcon) {
            node.appendChild(checkIcon);
          }
        }
        node.addEventListener('click', (function (selected) {
          return function (event) {
            event.stopPropagation();
            choose(selected);
          };
        })(option.value));
        optionsList.appendChild(node);
      }
      activeIndex = -1;
      for (var i = 0; i < list.length; i++) {
        if (list[i].value === value) {
          activeIndex = i;
          break;
        }
      }
    }
    updateTriggerLabel();
  }

  if (isSearchable && searchInput) {
    searchInput.addEventListener('input', function (event) {
      event.stopPropagation();
      searchQuery = this.value;
      if (clearBtn) {
        clearBtn.hidden = !searchQuery;
      }
      renderMenu();
      setActive(0);
    });
    if (clearBtn) {
      clearBtn.addEventListener('click', function (event) {
        event.stopPropagation();
        searchInput.value = '';
        searchQuery = '';
        clearBtn.hidden = true;
        renderMenu();
        searchInput.focus();
      });
    }
    searchInput.addEventListener('keydown', function (event) {
      if (event.key === 'Escape') {
        event.preventDefault();
        event.stopPropagation();
        closeMenu();
        trigger.focus();
        return;
      }
      if (event.key === 'Enter') {
        event.preventDefault();
        event.stopPropagation();
        var nodes = optionNodes();
        if (activeIndex >= 0 && nodes[activeIndex]) {
          choose(nodes[activeIndex].getAttribute('data-value'));
        } else if (nodes.length > 0) {
          choose(nodes[0].getAttribute('data-value'));
        }
        return;
      }
      if (event.key === 'ArrowDown') {
        event.preventDefault();
        setActive(activeIndex + 1);
        return;
      }
      if (event.key === 'ArrowUp') {
        event.preventDefault();
        setActive(activeIndex - 1);
        return;
      }
    });
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
    if (isSearchable) {
      searchQuery = '';
      if (searchInput) {
        searchInput.value = '';
      }
      if (clearBtn) {
        clearBtn.hidden = true;
      }
      renderMenu();
    }
    menu.hidden = false;
    container.classList.add('is-open');
    var fieldParent = container.closest('.field');
    if (fieldParent) {
      fieldParent.classList.add('has-open-select');
    }
    trigger.setAttribute('aria-expanded', 'true');
    menu.classList.remove('is-dropup');
    var rect = menu.getBoundingClientRect();
    var boundaryBottom = window.innerHeight;
    var modal = container.closest('.modal');
    if (modal) {
      var mRect = modal.getBoundingClientRect();
      boundaryBottom = Math.min(boundaryBottom, mRect.bottom);
    }
    if (rect.bottom > boundaryBottom - 8 && rect.top - rect.height > 8) {
      menu.classList.add('is-dropup');
    }
    setActive(indexOfValue(value));
    state.openSelect = component;
    if (isSearchable && searchInput) {
      setTimeout(function () {
        if (!menu.hidden && searchInput) {
          searchInput.focus();
        }
      }, 30);
    } else if (!menu.classList.contains('is-dropup') && modal) {
      setTimeout(function () {
        if (!menu.hidden && menu.scrollIntoView) {
          menu.scrollIntoView({ block: 'nearest', behavior: 'smooth' });
        }
      }, 30);
    }
  }

  function closeMenu() {
    menu.hidden = true;
    container.classList.remove('is-open');
    var fieldParent = container.closest('.field');
    if (fieldParent) {
      fieldParent.classList.remove('has-open-select');
    }
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
    if (onChange) {
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
    if (isSearchable && event.target === searchInput) {
      return;
    }
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

// ---------------------------------------------------------------------------
// MultiSelect: searchable multi-select dropdown for lists like allowed models.
// ---------------------------------------------------------------------------
function mountMultiSelect(config) {
  var container = config && config.container;
  if (!container) {
    return null;
  }
  var items = config.items || [];
  var selected = {};
  var initial = config.selected || [];
  for (var i = 0; i < initial.length; i++) {
    if (initial[i]) {
      selected[String(initial[i]).toLowerCase()] = true;
    }
  }
  var onChange = config.onChange;

  container.className = container.className ? container.className + ' multi-select' : 'multi-select';
  container.textContent = '';

  var trigger = document.createElement('button');
  trigger.type = 'button';
  trigger.className = 'multi-select-trigger';
  trigger.setAttribute('role', 'combobox');
  trigger.setAttribute('aria-haspopup', 'listbox');
  trigger.setAttribute('aria-expanded', 'false');

  var display = document.createElement('div');
  display.className = 'multi-select-display';

  var label = document.createElement('span');
  label.className = 'multi-select-label';
  display.appendChild(label);

  var badge = document.createElement('span');
  badge.className = 'multi-select-badge';
  badge.hidden = true;
  display.appendChild(badge);

  trigger.appendChild(display);

  var arrow = iconMarkup('chevron', 'multi-select-arrow');
  if (arrow) {
    trigger.appendChild(arrow);
  }

  var menu = document.createElement('div');
  menu.className = 'multi-select-menu';
  menu.setAttribute('role', 'listbox');
  menu.hidden = true;

  var searchWrap = document.createElement('div');
  searchWrap.className = 'multi-select-search-wrap';

  var searchIcon = iconMarkup('search', 'icon');
  if (searchIcon) {
    searchWrap.appendChild(searchIcon);
  }

  var searchInput = document.createElement('input');
  searchInput.type = 'text';
  searchInput.className = 'multi-select-search-input';
  searchInput.placeholder = t('key_form.quota_models_search');
  searchInput.autocomplete = 'off';
  searchInput.spellcheck = false;
  searchInput.setAttribute('data-bwignore', 'true');
  searchInput.setAttribute('data-lpignore', 'true');
  searchInput.setAttribute('data-1p-ignore', 'true');
  searchWrap.appendChild(searchInput);

  var clearBtn = document.createElement('button');
  clearBtn.type = 'button';
  clearBtn.className = 'multi-select-clear-btn';
  var clearIcon = iconMarkup('close', 'icon');
  if (clearIcon) {
    clearBtn.appendChild(clearIcon);
  }
  clearBtn.title = t('keys.filter_clear');
  clearBtn.hidden = true;
  searchWrap.appendChild(clearBtn);

  menu.appendChild(searchWrap);

  var bar = document.createElement('div');
  bar.className = 'multi-select-bar';

  var countSpan = document.createElement('span');
  countSpan.className = 'multi-select-count';
  bar.appendChild(countSpan);

  var barActions = document.createElement('div');
  barActions.className = 'multi-select-bar-actions';

  var selectAllBtn = document.createElement('button');
  selectAllBtn.type = 'button';
  selectAllBtn.className = 'btn-text';
  selectAllBtn.textContent = t('key_form.quota_select_all');
  barActions.appendChild(selectAllBtn);

  var clearAllBtn = document.createElement('button');
  clearAllBtn.type = 'button';
  clearAllBtn.className = 'btn-text';
  clearAllBtn.textContent = t('key_form.quota_clear_all');
  barActions.appendChild(clearAllBtn);

  bar.appendChild(barActions);
  menu.appendChild(bar);

  var optionsList = document.createElement('div');
  optionsList.className = 'multi-select-options';
  menu.appendChild(optionsList);

  var emptyDiv = document.createElement('div');
  emptyDiv.className = 'multi-select-empty';
  emptyDiv.textContent = t('key_form.quota_models_empty');
  emptyDiv.hidden = true;
  menu.appendChild(emptyDiv);

  container.appendChild(trigger);
  container.appendChild(menu);

  var filterText = '';

  function getSelectedList() {
    var out = [];
    for (var j = 0; j < items.length; j++) {
      var val = items[j].value;
      if (selected[String(val).toLowerCase()]) {
        out.push(val);
      }
    }
    return out;
  }

  function updateTriggerDisplay() {
    var sel = getSelectedList();
    if (sel.length === 0) {
      label.textContent = t('key_form.quota_models_placeholder');
      label.classList.add('is-placeholder');
      badge.hidden = true;
      trigger.title = '';
    } else {
      var text = sel.join(', ');
      label.textContent = text;
      label.classList.remove('is-placeholder');
      trigger.title = text;
      badge.textContent = String(sel.length);
      badge.hidden = sel.length <= 1;
    }
    countSpan.textContent = tf('key_form.quota_models_count', {
      selected: sel.length,
      total: items.length
    });
  }

  function renderOptions() {
    optionsList.textContent = '';
    var visibleCount = 0;
    var query = filterText.toLowerCase().trim();

    for (var k = 0; k < items.length; k++) {
      var item = items[k];
      var valStr = String(item.value);
      var labelStr = item.label === undefined ? valStr : String(item.label);
      var matches = !query || valStr.toLowerCase().indexOf(query) >= 0 || labelStr.toLowerCase().indexOf(query) >= 0;

      if (!matches) {
        continue;
      }
      visibleCount++;

      var opt = document.createElement('div');
      opt.className = 'multi-select-option';
      opt.setAttribute('role', 'option');
      opt.setAttribute('data-value', valStr);

      var isSel = selected[valStr.toLowerCase()] === true;
      if (isSel) {
        opt.classList.add('is-selected');
        opt.setAttribute('aria-selected', 'true');
      } else {
        opt.setAttribute('aria-selected', 'false');
      }

      var chk = document.createElement('span');
      chk.className = 'multi-select-checkbox';
      var checkIco = iconMarkup('check', 'multi-select-check-icon');
      if (checkIco) {
        chk.appendChild(checkIco);
      }
      opt.appendChild(chk);

      var txt = document.createElement('span');
      txt.className = 'multi-select-text';
      txt.textContent = labelStr;
      opt.appendChild(txt);

      (function (v) {
        opt.addEventListener('click', function (e) {
          e.stopPropagation();
          toggleItem(v);
        });
      })(valStr);

      optionsList.appendChild(opt);
    }

    emptyDiv.hidden = visibleCount > 0;
    updateTriggerDisplay();
  }

  function toggleItem(val) {
    var key = String(val).toLowerCase();
    if (selected[key]) {
      delete selected[key];
    } else {
      selected[key] = true;
    }
    renderOptions();
    if (onChange) {
      onChange(getSelectedList());
    }
  }

  function openMenu() {
    if (state.openSelect && state.openSelect !== component) {
      state.openSelect.close();
    }
    menu.hidden = false;
    container.classList.add('is-open');
    trigger.setAttribute('aria-expanded', 'true');

    menu.classList.remove('is-dropup');
    var rect = trigger.getBoundingClientRect();
    var modal = container.closest ? container.closest('.modal') : null;
    var bottomSpace = window.innerHeight - rect.bottom;
    var topSpace = rect.top;

    if (modal) {
      var modalRect = modal.getBoundingClientRect();
      bottomSpace = modalRect.bottom - rect.bottom;
      topSpace = rect.top - modalRect.top;
    }

    if (bottomSpace < 240 && topSpace > 180) {
      menu.classList.add('is-dropup');
    }

    state.openSelect = component;

    if (!menu.classList.contains('is-dropup') && modal) {
      setTimeout(function () {
        if (!menu.hidden && menu.scrollIntoView) {
          menu.scrollIntoView({ block: 'nearest', behavior: 'smooth' });
        }
      }, 30);
    }

    setTimeout(function () {
      searchInput.focus();
    }, 50);
  }

  function closeMenu() {
    menu.hidden = true;
    container.classList.remove('is-open');
    trigger.setAttribute('aria-expanded', 'false');
    if (state.openSelect === component) {
      state.openSelect = null;
    }
  }

  trigger.addEventListener('click', function (e) {
    e.stopPropagation();
    if (menu.hidden) {
      openMenu();
    } else {
      closeMenu();
    }
  });

  menu.addEventListener('click', function (e) {
    e.stopPropagation();
  });

  searchInput.addEventListener('input', function () {
    filterText = searchInput.value;
    clearBtn.hidden = !filterText;
    renderOptions();
    if (!menu.classList.contains('is-dropup') && container.closest) {
      var m = container.closest('.modal');
      if (m && menu.scrollIntoView) {
        menu.scrollIntoView({ block: 'nearest' });
      }
    }
  });

  clearBtn.addEventListener('click', function (e) {
    e.stopPropagation();
    searchInput.value = '';
    filterText = '';
    clearBtn.hidden = true;
    renderOptions();
    searchInput.focus();
  });

  selectAllBtn.addEventListener('click', function (e) {
    e.stopPropagation();
    for (var a = 0; a < items.length; a++) {
      selected[String(items[a].value).toLowerCase()] = true;
    }
    renderOptions();
    if (onChange) {
      onChange(getSelectedList());
    }
  });

  clearAllBtn.addEventListener('click', function (e) {
    e.stopPropagation();
    selected = {};
    renderOptions();
    if (onChange) {
      onChange(getSelectedList());
    }
  });

  container.addEventListener('keydown', function (e) {
    if (e.key === 'Escape') {
      if (!menu.hidden) {
        e.preventDefault();
        e.stopPropagation();
        closeMenu();
        trigger.focus();
      }
    }
  });

  var component = {
    element: container,
    getSelected: function () {
      return getSelectedList();
    },
    setSelected: function (values) {
      selected = {};
      var arr = Array.isArray(values) ? values : [];
      for (var s = 0; s < arr.length; s++) {
        if (arr[s]) {
          selected[String(arr[s]).toLowerCase()] = true;
        }
      }
      renderOptions();
      if (onChange) {
        onChange(getSelectedList());
      }
    },
    setItems: function (nextItems) {
      items = nextItems || [];
      renderOptions();
    },
    refresh: function () {
      searchInput.placeholder = t('key_form.quota_models_search');
      selectAllBtn.textContent = t('key_form.quota_select_all');
      clearAllBtn.textContent = t('key_form.quota_clear_all');
      emptyDiv.textContent = t('key_form.quota_models_empty');
      renderOptions();
    },
    close: closeMenu
  };

  state.selects.push(component);
  renderOptions();
  return component;
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
