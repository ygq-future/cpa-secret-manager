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

// appendLabelParts renders structured option text without innerHTML. Each part
// carries plain text and an optional class, so callers can color prices or
// truncate a name while keeping a trailing value visible.
function appendLabelParts(container, parts) {
  container.textContent = '';
  for (var index = 0; index < parts.length; index++) {
    var part = parts[index];
    if (!part) {
      continue;
    }
    var span = document.createElement('span');
    if (part.className) {
      span.className = part.className;
    }
    span.textContent = part.text === undefined ? '' : String(part.text);
    container.appendChild(span);
  }
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

  var clearAction = null;
  if (config.clearable) {
    clearAction = document.createElement('button');
    clearAction.type = 'button';
    clearAction.className = 'select-clear-btn select-clear-action';
    clearAction.hidden = true;
    if (config.clearTitleKey) {
      clearAction.title = t(config.clearTitleKey);
      clearAction.setAttribute('aria-label', t(config.clearTitleKey));
    }
    var clearActionIcon = iconMarkup('close', 'icon');
    if (clearActionIcon) {
      clearAction.appendChild(clearActionIcon);
    }
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
  if (clearAction) {
    container.appendChild(clearAction);
  }
  container.appendChild(menu);

  var activeIndex = -1;

  function optionLabel(option) {
    if (option.i18n) {
      return t(option.i18n);
    }
    return option.label === undefined ? String(option.value) : option.label;
  }

  function optionTriggerLabel(option) {
    if (option.triggerLabel !== undefined) {
      return option.triggerLabel;
    }
    return optionLabel(option);
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
    var hasValue = index >= 0;
    if (hasValue) {
      if (options[index].triggerParts && options[index].triggerParts.length) {
        appendLabelParts(label, options[index].triggerParts);
      } else {
        label.textContent = optionTriggerLabel(options[index]);
      }
      label.classList.remove('is-placeholder');
    } else if (config.placeholderKey || config.placeholder) {
      label.textContent = config.placeholderKey ? t(config.placeholderKey) : config.placeholder;
      label.classList.add('is-placeholder');
    } else {
      label.textContent = '';
      label.classList.remove('is-placeholder');
    }
    if (!config.titleKey) {
      trigger.title = hasValue && (options[index].label !== undefined || options[index].i18n)
        ? optionLabel(options[index])
        : '';
    }
    var showClear = Boolean(clearAction && hasValue);
    container.classList.toggle('has-clear', showClear);
    if (clearAction) {
      clearAction.hidden = !showClear;
    }
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
      empty.textContent = config.emptyKey ? t(config.emptyKey) : t('key_form.quota_models_empty');
      optionsList.appendChild(empty);
      activeIndex = -1;
    } else {
      for (var index = 0; index < list.length; index++) {
        var option = list[index];
        var isSelected = !option.disabled && option.value === value;
        var node = document.createElement('div');
        node.className = 'select-option' + (isSelected ? ' is-selected' : '') + (option.disabled ? ' is-disabled' : '');
        node.setAttribute('role', 'option');
        node.setAttribute('aria-selected', isSelected ? 'true' : 'false');
        if (option.disabled) {
          node.setAttribute('aria-disabled', 'true');
        }
        node.setAttribute('data-value', option.value);
        if (option.label !== undefined || option.i18n) {
          node.title = optionLabel(option);
        }
        var labelSpan = document.createElement('span');
        labelSpan.className = 'select-option-label';
        if (option.labelParts && option.labelParts.length) {
          appendLabelParts(labelSpan, option.labelParts);
        } else {
          labelSpan.textContent = optionLabel(option);
        }
        node.appendChild(labelSpan);
        if (option.trailingParts && option.trailingParts.length) {
          var trailingSpan = document.createElement('span');
          trailingSpan.className = 'select-option-trailing';
          appendLabelParts(trailingSpan, option.trailingParts);
          node.appendChild(trailingSpan);
        }
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
        moveActive(1);
        return;
      }
      if (event.key === 'ArrowUp') {
        event.preventDefault();
        moveActive(-1);
        return;
      }
    });
  }

  function setActive(index) {
    var nodes = optionNodes();
    if (!nodes.length) {
      activeIndex = -1;
      return;
    }
    activeIndex = ((index % nodes.length) + nodes.length) % nodes.length;
    for (var position = 0; position < nodes.length; position++) {
      nodes[position].classList.toggle('is-active', position === activeIndex);
    }
    if (nodes[activeIndex].scrollIntoView) {
      nodes[activeIndex].scrollIntoView({ block: 'nearest' });
    }
  }

  function clearActive() {
    activeIndex = -1;
    var nodes = optionNodes();
    for (var position = 0; position < nodes.length; position++) {
      nodes[position].classList.remove('is-active');
    }
  }

  function moveActive(delta) {
    var nodes = optionNodes();
    if (!nodes.length) {
      return;
    }
    if (activeIndex < 0) {
      setActive(delta > 0 ? 0 : nodes.length - 1);
      return;
    }
    setActive(activeIndex + delta);
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
    var selectedIndex = indexOfValue(value);
    if (selectedIndex >= 0) {
      setActive(selectedIndex);
    } else {
      clearActive();
    }
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
    var picked = indexOfValue(next);
    if (picked >= 0 && options[picked].disabled) {
      return;
    }
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

  if (clearAction) {
    clearAction.addEventListener('click', function (event) {
      event.preventDefault();
      event.stopPropagation();
      value = '';
      renderMenu();
      closeMenu();
      if (onChange) {
        onChange('');
      }
      trigger.focus();
    });
  }

  container.addEventListener('keydown', function (event) {
    var key = event.key;
    if (clearAction && event.target === clearAction) {
      return;
    }
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
      moveActive(1);
    } else if (key === 'ArrowUp') {
      moveActive(-1);
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

      if (item.tags && item.tags.length) {
        var meta = document.createElement('span');
        meta.className = 'multi-select-meta';
        for (var t = 0; t < item.tags.length; t++) {
          var tag = document.createElement('span');
          tag.className = 'price-tier-tag' + (item.tags[t].tone ? ' is-' + item.tags[t].tone : '');
          tag.textContent = item.tags[t].text;
          meta.appendChild(tag);
        }
        opt.appendChild(meta);
      }

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

// ---------------------------------------------------------------------------
// DateTime picker: a themed date + time popover. Native datetime controls
// cannot follow the host theme, so the calendar grid, the time columns and the
// presets are all rendered with the shared tokens. The committed value is an
// ISO instant; an empty string means "no expiry".
// ---------------------------------------------------------------------------
function mountDateTimePicker(config) {
  var container = config && config.container;
  if (!container) {
    return null;
  }
  var onChange = config.onChange;
  var mode = config.mode === 'past' ? 'past' : 'future';
  var placeholderKey = config.placeholderKey || 'datetime.placeholder';
  var clearTitleKey = config.clearTitleKey || 'datetime.clear';
  var value = typeof config.value === 'string' ? config.value : '';
  var draft = null;
  var baseMonth = null;
  var open = false;
  var presetSpecs = mode === 'past'
    ? [
        { key: 'datetime.past_month', minutes: 43200 },
        { key: 'datetime.past_week', minutes: 10080 },
        { key: 'datetime.past_day', minutes: 1440 },
        { key: 'datetime.past_6hours', minutes: 360 },
        { key: 'datetime.past_hour', minutes: 60 }
      ]
    : [
        { key: 'datetime.preset_now', minutes: 1, now: true },
        { key: 'datetime.preset_hour', minutes: 60 },
        { key: 'datetime.preset_day', minutes: 1440 },
        { key: 'datetime.preset_week', minutes: 10080 },
        { key: 'datetime.preset_month', minutes: 43200 }
      ];
  container.className = 'datetime-field';
  container.textContent = '';

  var control = document.createElement('div');
  control.className = 'datetime-control';
  var trigger = document.createElement('button');
  trigger.type = 'button';
  trigger.className = 'datetime-trigger';
  trigger.setAttribute('aria-haspopup', 'dialog');
  trigger.setAttribute('aria-expanded', 'false');
  var glyph = iconMarkup('calendar', 'datetime-icon');
  if (glyph) {
    trigger.appendChild(glyph);
  }
  var label = document.createElement('span');
  label.className = 'datetime-label';
  trigger.appendChild(label);
  control.appendChild(trigger);

  var clearBtn = document.createElement('button');
  clearBtn.type = 'button';
  clearBtn.className = 'datetime-clear';
  clearBtn.hidden = true;
  var clearGlyph = iconMarkup('close', '');
  if (clearGlyph) {
    clearBtn.appendChild(clearGlyph);
  }
  control.appendChild(clearBtn);
  container.appendChild(control);

  var popover = document.createElement('div');
  popover.className = 'datetime-popover';
  popover.hidden = true;
  popover.tabIndex = -1;
  popover.setAttribute('role', 'dialog');
  document.body.appendChild(popover);

  var presetsRow = document.createElement('div');
  presetsRow.className = 'datetime-presets';
  popover.appendChild(presetsRow);

  var presetButtons = [];
  for (var presetIndex = 0; presetIndex < presetSpecs.length; presetIndex++) {
    presetButtons.push(buildPresetButton(presetSpecs[presetIndex]));
  }
  function buildPresetButton(spec) {
    var button = document.createElement('button');
    button.type = 'button';
    button.className = 'pill';
    button.textContent = t(spec.key);
    if (spec.now) {
      button.title = t('datetime.preset_now_title');
      button.setAttribute('aria-label', t('datetime.preset_now_title'));
    }
    button.addEventListener('click', function () {
      // A preset is a quick draft edit, not a commit: keep the popover open so
      // the operator can still tweak the day, hour, or minute before confirming.
      var delta = spec.minutes * 60000;
      var at = new Date(mode === 'past' ? Date.now() - delta : Date.now() + delta);
      at.setSeconds(0, 0);
      draft = at;
      baseMonth = new Date(draft.getFullYear(), draft.getMonth(), 1);
      renderCalendar();
      syncTime();
    });
    presetsRow.appendChild(button);
    return button;
  }

  var calendarWrap = document.createElement('div');
  calendarWrap.className = 'datetime-calendar';
  popover.appendChild(calendarWrap);

  var timeRow = document.createElement('div');
  timeRow.className = 'datetime-time';
  var timeHeading = document.createElement('span');
  timeHeading.className = 'datetime-time-label';
  timeHeading.setAttribute('data-time-label', 'datetime.time');
  timeHeading.textContent = t('datetime.time');
  timeRow.appendChild(timeHeading);
  var timeFields = document.createElement('div');
  timeFields.className = 'time-fields';
  timeRow.appendChild(timeFields);
  var hourInput = buildTimeStepper('datetime.hour', 'datetime.hour_unit', 23, setHour);
  var timeSeparator = document.createElement('span');
  timeSeparator.className = 'time-separator';
  timeSeparator.textContent = ':';
  timeFields.appendChild(timeSeparator);
  var minuteInput = buildTimeStepper('datetime.minute', 'datetime.minute_unit', 59, setMinute);
  popover.appendChild(timeRow);

  var hint = document.createElement('p');
  hint.className = 'datetime-hint';
  hint.hidden = true;
  popover.appendChild(hint);

  var footer = document.createElement('div');
  footer.className = 'datetime-footer';
  var footerClear = document.createElement('button');
  footerClear.type = 'button';
  footerClear.className = 'btn btn-secondary';
  footerClear.addEventListener('click', function () {
    commit('');
  });
  var footerConfirm = document.createElement('button');
  footerConfirm.type = 'button';
  footerConfirm.className = 'btn btn-primary';
  footerConfirm.addEventListener('click', function () {
    if (!draft || (mode === 'past' ? draft.getTime() > Date.now() : draft.getTime() <= Date.now())) {
      return;
    }
    commit(draft.toISOString());
  });
  footer.appendChild(footerClear);
  footer.appendChild(footerConfirm);
  popover.appendChild(footer);

  function buildTimeStepper(labelKey, unitKey, max, onPick) {
    var wrap = document.createElement('div');
    wrap.className = 'time-stepper';
    var input = document.createElement('input');
    input.type = 'text';
    input.className = 'time-value';
    input.setAttribute('inputmode', 'numeric');
    input.setAttribute('maxlength', '2');
    input.setAttribute('data-time-input', labelKey);
    input.setAttribute('aria-label', t(labelKey));
    var unit = document.createElement('span');
    unit.className = 'time-unit';
    unit.setAttribute('data-time-unit', unitKey);
    unit.textContent = t(unitKey);
    var buttons = document.createElement('div');
    buttons.className = 'time-buttons';
    buttons.appendChild(buildTimeStepButton('is-up', 'datetime.increment'));
    buttons.appendChild(buildTimeStepButton('is-down', 'datetime.decrement'));
    wrap.appendChild(input);
    wrap.appendChild(unit);
    wrap.appendChild(buttons);
    timeFields.appendChild(wrap);

    function current() {
      var parsed = parseInt(input.value, 10);
      return isNaN(parsed) ? 0 : parsed;
    }

    function apply(next) {
      var normalized = Math.min(max, Math.max(0, next));
      input.value = pad2(normalized);
      onPick(normalized);
    }

    function step(delta) {
      var next = current() + delta;
      if (next > max) {
        next = 0;
      } else if (next < 0) {
        next = max;
      }
      apply(next);
    }

    input.addEventListener('input', function () {
      input.value = input.value.replace(/[^0-9]/g, '').slice(0, 2);
    });
    input.addEventListener('change', function () {
      apply(current());
    });
    input.addEventListener('blur', function () {
      apply(current());
    });
    input.addEventListener('keydown', function (event) {
      if (event.key === 'ArrowUp') {
        event.preventDefault();
        step(1);
      } else if (event.key === 'ArrowDown') {
        event.preventDefault();
        step(-1);
      }
    });
    wrap.addEventListener('wheel', function (event) {
      if (!event.deltaY) {
        return;
      }
      event.preventDefault();
      step(event.deltaY < 0 ? 1 : -1);
    }, { passive: false });

    var buttonsList = buttons.querySelectorAll('.time-step');
    buttonsList[0].addEventListener('click', function () {
      step(1);
    });
    buttonsList[1].addEventListener('click', function () {
      step(-1);
    });
    return input;
  }

  function buildTimeStepButton(className, labelKey) {
    var button = document.createElement('button');
    button.type = 'button';
    button.className = 'time-step ' + className;
    button.setAttribute('data-step-label', labelKey);
    button.setAttribute('aria-label', t(labelKey));
    button.appendChild(iconMarkup('chevron', ''));
    return button;
  }

  function pad2(number) {
    return number < 10 ? '0' + number : '' + number;
  }

  function formatLabel(at) {
    return at.getFullYear() + '/' + pad2(at.getMonth() + 1) + '/' + pad2(at.getDate()) + ' ' + pad2(at.getHours()) + ':' + pad2(at.getMinutes());
  }

  function parsedValue() {
    if (!value) {
      return null;
    }
    var at = new Date(value);
    return isNaN(at.getTime()) ? null : at;
  }

  function updateLabel() {
    var at = parsedValue();
    label.textContent = at ? formatLabel(at) : t(placeholderKey);
    label.classList.toggle('is-placeholder', !at);
    control.classList.toggle('has-value', Boolean(at));
    clearBtn.hidden = !at;
    clearBtn.title = t(clearTitleKey);
    clearBtn.setAttribute('aria-label', t(clearTitleKey));
  }

  function setValue(next) {
    value = typeof next === 'string' ? next : '';
    updateLabel();
  }

  function defaultDraft() {
    var at = parsedValue();
    if (at) {
      return at;
    }
    var offset = mode === 'past' ? -3600000 : 86400000;
    var initial = new Date(Date.now() + offset);
    initial.setSeconds(0, 0);
    return initial;
  }

  function syncTime() {
    hourInput.value = pad2(draft.getHours());
    minuteInput.value = pad2(draft.getMinutes());
    var valid = mode === 'past' ? draft.getTime() <= Date.now() : draft.getTime() > Date.now();
    footerConfirm.disabled = !valid;
    hint.hidden = valid;
    hint.textContent = valid ? '' : t(mode === 'past' ? 'datetime.future' : 'datetime.past');
    if (open) {
      positionPopover();
    }
  }

  function setHour(hour) {
    draft.setHours(hour, draft.getMinutes(), 0, 0);
    syncTime();
  }

  function setMinute(minute) {
    draft.setMinutes(minute, 0, 0);
    syncTime();
  }

  function monthTitle(monthDate) {
    if (state.language === 'en-US') {
      try {
        return new Intl.DateTimeFormat('en-US', { month: 'long', year: 'numeric' }).format(monthDate);
      } catch (error) {}
    }
    return monthDate.getFullYear() + '年' + (monthDate.getMonth() + 1) + '月';
  }

  function renderCalendar() {
    calendarWrap.textContent = '';

    var head = document.createElement('div');
    head.className = 'calendar-head';
    var prev = document.createElement('button');
    prev.type = 'button';
    prev.className = 'btn-icon';
    prev.title = t('range.previous_month');
    prev.setAttribute('aria-label', t('range.previous_month'));
    prev.appendChild(iconMarkup('prev', ''));
    prev.addEventListener('click', function () {
      baseMonth = new Date(baseMonth.getFullYear(), baseMonth.getMonth() - 1, 1);
      renderCalendar();
    });
    var title = document.createElement('span');
    title.textContent = monthTitle(baseMonth);
    var next = document.createElement('button');
    next.type = 'button';
    next.className = 'btn-icon';
    next.title = t('range.next_month');
    next.setAttribute('aria-label', t('range.next_month'));
    next.appendChild(iconMarkup('next', ''));
    next.addEventListener('click', function () {
      baseMonth = new Date(baseMonth.getFullYear(), baseMonth.getMonth() + 1, 1);
      renderCalendar();
    });
    head.appendChild(prev);
    head.appendChild(title);
    head.appendChild(next);
    calendarWrap.appendChild(head);

    var grid = document.createElement('div');
    grid.className = 'calendar-grid';
    var weekKeys = ['range.week_sun', 'range.week_mon', 'range.week_tue', 'range.week_wed', 'range.week_thu', 'range.week_fri', 'range.week_sat'];
    for (var weekIndex = 0; weekIndex < weekKeys.length; weekIndex++) {
      var week = document.createElement('div');
      week.className = 'calendar-week';
      week.textContent = t(weekKeys[weekIndex]);
      grid.appendChild(week);
    }

    var today = new Date();
    var todayMidnight = new Date(today.getFullYear(), today.getMonth(), today.getDate()).getTime();
    var firstDay = new Date(baseMonth.getFullYear(), baseMonth.getMonth(), 1).getDay();
    var totalDays = new Date(baseMonth.getFullYear(), baseMonth.getMonth() + 1, 0).getDate();
    for (var blank = 0; blank < firstDay; blank++) {
      var empty = document.createElement('div');
      empty.className = 'calendar-day empty';
      grid.appendChild(empty);
    }
    for (var day = 1; day <= totalDays; day++) {
      var cell = document.createElement('button');
      cell.type = 'button';
      cell.className = 'calendar-day datetime-day';
      cell.textContent = '' + day;
      var dateMidnight = new Date(baseMonth.getFullYear(), baseMonth.getMonth(), day).getTime();
      var disabledDay = mode === 'past' ? dateMidnight > todayMidnight : dateMidnight < todayMidnight;
      if (disabledDay) {
        cell.classList.add('is-disabled');
        cell.disabled = true;
      } else {
        if (dateMidnight === new Date(draft.getFullYear(), draft.getMonth(), draft.getDate()).getTime()) {
          cell.classList.add('is-selected');
        }
        cell.addEventListener('click', (function (selectedDay) {
          return function () {
            draft.setFullYear(baseMonth.getFullYear(), baseMonth.getMonth(), selectedDay);
            draft.setSeconds(0, 0);
            renderCalendar();
            syncTime();
          };
        })(day));
      }
      grid.appendChild(cell);
    }
    calendarWrap.appendChild(grid);
  }

  function close() {
    if (!open) {
      return;
    }
    open = false;
    popover.hidden = true;
    container.classList.remove('is-open');
    trigger.setAttribute('aria-expanded', 'false');
    if (state.openDateTime === api) {
      state.openDateTime = null;
    }
  }

  function positionPopover() {
    if (!open) {
      return;
    }
    var rect = trigger.getBoundingClientRect();
    var popRect = popover.getBoundingClientRect();
    var left = rect.left;
    if (left + popRect.width > window.innerWidth - 8) {
      left = window.innerWidth - popRect.width - 8;
    }
    if (left < 8) {
      left = 8;
    }
    var gap = 6;
    var spaceBelow = window.innerHeight - rect.bottom - gap;
    var spaceAbove = rect.top - gap;
    var top;
    if (popRect.height <= spaceBelow) {
      top = rect.bottom + gap;
    } else if (popRect.height <= spaceAbove) {
      top = rect.top - popRect.height - gap;
    } else if (spaceBelow >= spaceAbove) {
      top = Math.max(8, window.innerHeight - popRect.height - 8);
    } else {
      top = 8;
    }
    popover.style.setProperty('left', left + 'px');
    popover.style.setProperty('top', top + 'px');
  }

  function openPicker() {
    if (open) {
      return;
    }
    if (state.openDateTime && state.openDateTime !== api) {
      state.openDateTime.close();
    }
    draft = defaultDraft();
    baseMonth = new Date(draft.getFullYear(), draft.getMonth(), 1);
    renderCalendar();
    popover.hidden = false;
    open = true;
    container.classList.add('is-open');
    trigger.setAttribute('aria-expanded', 'true');
    syncTime();
    popover.focus();
    state.openDateTime = api;
  }

  function commit(next) {
    var changed = (next || '') !== value;
    setValue(next || '');
    close();
    if (changed && onChange) {
      onChange(value);
    }
  }

  trigger.addEventListener('click', function (event) {
    event.stopPropagation();
    if (open) {
      close();
    } else {
      openPicker();
    }
  });

  clearBtn.addEventListener('click', function (event) {
    event.stopPropagation();
    commit('');
  });

  // Calendar cells re-render on click, so the original target can be detached
  // from the DOM before the document outside-click listener runs. Keeping the
  // click inside the popover prevents that listener from closing a live picker.
  popover.addEventListener('click', function (event) {
    event.stopPropagation();
  });

  popover.addEventListener('keydown', function (event) {
    if (event.key === 'Escape' && open) {
      event.stopPropagation();
      close();
    }
  });

  document.addEventListener('click', function (event) {
    if (open && !container.contains(event.target) && !popover.contains(event.target)) {
      close();
    }
  });
  window.addEventListener('resize', positionPopover);
  window.addEventListener('scroll', positionPopover, true);

  var api = {
    getValue: function () {
      return value;
    },
    setValue: setValue,
    open: openPicker,
    close: close,
    refresh: function () {
      updateLabel();
      for (var index = 0; index < presetButtons.length; index++) {
        presetButtons[index].textContent = t(presetSpecs[index].key);
        if (presetSpecs[index].now) {
          presetButtons[index].title = t('datetime.preset_now_title');
          presetButtons[index].setAttribute('aria-label', t('datetime.preset_now_title'));
        }
      }
      footerClear.textContent = t('datetime.clear_action');
      footerConfirm.textContent = t('datetime.confirm');
      var headings = popover.querySelectorAll('[data-time-label]');
      for (var headingIndex = 0; headingIndex < headings.length; headingIndex++) {
        headings[headingIndex].textContent = t(headings[headingIndex].getAttribute('data-time-label'));
      }
      var timeInputs = popover.querySelectorAll('[data-time-input]');
      for (var inputIndex = 0; inputIndex < timeInputs.length; inputIndex++) {
        timeInputs[inputIndex].setAttribute('aria-label', t(timeInputs[inputIndex].getAttribute('data-time-input')));
      }
      var units = popover.querySelectorAll('[data-time-unit]');
      for (var unitIndex = 0; unitIndex < units.length; unitIndex++) {
        units[unitIndex].textContent = t(units[unitIndex].getAttribute('data-time-unit'));
      }
      var stepButtons = popover.querySelectorAll('[data-step-label]');
      for (var stepIndex = 0; stepIndex < stepButtons.length; stepIndex++) {
        stepButtons[stepIndex].setAttribute('aria-label', t(stepButtons[stepIndex].getAttribute('data-step-label')));
      }
      if (open) {
        renderCalendar();
        syncTime();
      }
    }
  };

  updateLabel();
  footerClear.textContent = t('datetime.clear_action');
  footerConfirm.textContent = t('datetime.confirm');
  state.selects.push(api);
  return api;
}

function closeOpenDateTime() {
  if (state.openDateTime) {
    state.openDateTime.close();
  }
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
