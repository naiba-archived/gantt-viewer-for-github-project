(function () {
    const changeThemeSelect = document.getElementById('change-theme');
    const themes = ["auto", "light", "dark", "cupcake", "bumblebee", "emerald", "corporate", "synthwave", "retro", "cyberpunk", "valentine", "halloween", "garden", "forest", "aqua", "lofi", "pastel", "fantasy", "wireframe", "black", "luxury", "dracula", "cmyk", "autumn", "business", "acid", "lemonade", "night", "coffee", "winter"]
    themes.forEach(t => {
        changeThemeSelect.innerHTML += `<option value="${t}" ${t == Cookies.get('theme') ? 'selected' : ''}>${t}</option>`
    })
    changeThemeSelect.addEventListener('change', e => {
        Cookies.set('theme', e.target.value)
        window.location.reload()
    })

    const tasks = [
        {
            id: 'Task 1',
            name: 'Your gantt viewer',
            start: '2016-12-28',
            end: '2016-12-31',
            progress: 100,
        },
        {
            id: 'Task 2',
            name: 'is',
            start: '2017-01-01',
            end: '2017-01-02',
            progress: 20,
            dependencies: 'Task 1'
        },
        {
            id: 'Task 3',
            name: 'comming',
            start: '2017-01-03',
            end: '2017-01-04',
            progress: 0,
            dependencies: 'Task 2'
        },
        {
            id: 'Task 4',
            name: 'blabla',
            start: '2017-01-03',
            end: '2017-01-04',
            progress: 0,
        },
    ]
    if (document.getElementById("home-gantt-demo")) {
        const homeGantt = new Gantt("#home-gantt-demo", tasks)
        homeGantt.$svg.setAttribute('height', homeGantt.$svg.getAttribute('height') - 100);
    }
})()

function logout() {
    if (confirm('Are you sure you want to logout?')) {
        post('/logout', {})
    }
}

function post(path, params, method = 'post') {
    const form = document.createElement('form');
    form.method = method;
    form.action = path;

    for (const key in params) {
        if (params.hasOwnProperty(key)) {
            const hiddenField = document.createElement('input');
            hiddenField.type = 'hidden';
            hiddenField.name = key;
            hiddenField.value = params[key];

            form.appendChild(hiddenField);
        }
    }

    document.body.appendChild(form);
    form.submit();
}