
// Sidebar
const sidebar = document.querySelector('.sidebar');
const toggle = document.querySelector('.nav-toggle');
const MOBILE = 640;   // must match the media query

let collapsed;
try {
    const saved = localStorage.getItem('navCollapsed');
    collapsed = saved === null ? window.innerWidth <= MOBILE : saved === 'true';
} catch (e) {
    collapsed = window.innerWidth <= MOBILE;
}

document.body.classList.toggle('nav-collapsed', collapsed);
toggle.setAttribute('aria-expanded', String(!collapsed));

toggle.addEventListener('click', () => {
    const isCollapsed = document.body.classList.toggle('nav-collapsed');
    toggle.setAttribute('aria-expanded', String(!isCollapsed));
    try {
        localStorage.setItem('navCollapsed', isCollapsed);
    } catch (e) {}
});
