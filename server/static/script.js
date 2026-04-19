const state = {
    token: localStorage.getItem('wb_token') || '',
    user: null,
};

const authSection = document.getElementById('authSection');
const appSection = document.getElementById('appSection');
const sessionBox = document.getElementById('sessionBox');

document.getElementById('loginForm').addEventListener('submit', async (event) => {
    event.preventDefault();
    await authenticate('/api/auth/login', {
        username: document.getElementById('loginUsername').value,
        password: document.getElementById('loginPassword').value,
    }, 'loginStatus');
});

document.getElementById('registerForm').addEventListener('submit', async (event) => {
    event.preventDefault();
    await authenticate('/api/auth/register', {
        username: document.getElementById('registerUsername').value,
        password: document.getElementById('registerPassword').value,
    }, 'registerStatus');
});

document.getElementById('logoutBtn').addEventListener('click', () => {
    state.token = '';
    state.user = null;
    localStorage.removeItem('wb_token');
    renderSession();
});

document.querySelectorAll('.tab').forEach((button) => {
    button.addEventListener('click', () => activateTab(button.dataset.tab));
});

document.getElementById('filtersForm').addEventListener('submit', async (event) => {
    event.preventDefault();
    await loadOrders();
});

document.getElementById('loadOrdersBtn').addEventListener('click', loadOrders);
document.getElementById('loadAggregateBtn').addEventListener('click', loadAggregate);

document.getElementById('detailForm').addEventListener('submit', async (event) => {
    event.preventDefault();
    await loadOrderDetail(document.getElementById('detailOrderUid').value.trim());
});

init();

async function init() {
    renderSession();
    if (!state.token) {
        return;
    }

    try {
        state.user = await apiFetch('/api/auth/me');
        renderSession();
        setStatus(statusId, 'Авторизация прошла, загружаю данные...', 'ok');
        await refreshDashboard();
        setStatus(statusId, 'Готово', 'ok');
    } catch (error) {
        state.token = '';
        localStorage.removeItem('wb_token');
        renderSession();
    }
}

async function authenticate(url, payload, statusId) {
    setStatus(statusId, 'Загрузка...');
    try {
        const response = await fetch(url, {
            method: 'POST',
            headers: {'Content-Type': 'application/json'},
            body: JSON.stringify(payload),
        });
        const data = await readJSON(response);
        if (!response.ok) {
            throw new Error(data.error || `HTTP ${response.status}`);
        }

        state.token = data.token;
        state.user = data.user;
        localStorage.setItem('wb_token', state.token);
        setStatus(statusId, 'Готово', 'ok');
        renderSession();
        await loadFilterValues();
        await loadOrders();
        await loadAggregate();
    } catch (error) {
        setStatus(statusId, error.message, 'error');
    }
}

function renderSession() {
    const authenticated = Boolean(state.token);
    authSection.classList.toggle('hidden', authenticated);
    appSection.classList.toggle('hidden', !authenticated);
    sessionBox.textContent = authenticated && state.user ? `Пользователь: ${state.user.username}` : '';
}

function activateTab(panelId) {
    document.querySelectorAll('.tab').forEach((button) => {
        button.classList.toggle('active', button.dataset.tab === panelId);
    });
    ['ordersPanel', 'aggregatePanel', 'detailPanel'].forEach((id) => {
        document.getElementById(id).classList.toggle('hidden', id !== panelId);
    });
}

async function loadFilterValues() {
    const values = await apiFetch('/api/orders/filter-values');
    fillSelect('filterPhone', values.phones);
    fillSelect('filterCustomer', values.customers);
    fillSelect('filterDelivery', values.delivery_services);
    fillSelect('filterBank', values.banks);
    fillSelect('filterCurrency', values.currencies);
    fillSelect('filterCity', values.cities);
    fillSelect('filterRegion', values.regions);
    fillSelect('filterShardkey', values.shardkeys);
}

async function loadOrders() {
    setStatus('ordersStatus', 'Загрузка...');
    try {
        const data = await apiFetch(`/api/orders?${buildFilterParams()}`);
        renderOrders(data.orders || []);
        setStatus('ordersStatus', `Заказов: ${(data.orders || []).length}`, 'ok');
    } catch (error) {
        setStatus('ordersStatus', error.message, 'error');
    }
}

async function loadAggregate() {
    setStatus('aggregateStatus', 'Загрузка...');
    try {
        const groupBy = document.getElementById('groupBy').value;
        const data = await apiFetch(`/api/orders/aggregate?group_by=${encodeURIComponent(groupBy)}&${buildFilterParams()}`);
        renderAggregations(data.items || []);
        setStatus('aggregateStatus', `Групп: ${(data.items || []).length}`, 'ok');
    } catch (error) {
        setStatus('aggregateStatus', error.message, 'error');
    }
}

async function loadOrderDetail(orderUid) {
    if (!orderUid) {
        setStatus('detailStatus', 'Укажите Order UID', 'error');
        return;
    }

    activateTab('detailPanel');
    setStatus('detailStatus', 'Загрузка...');
    try {
        const order = await apiFetch(`/api/order/${encodeURIComponent(orderUid)}`);
        renderOrderDetail(order);
        setStatus('detailStatus', 'Готово', 'ok');
    } catch (error) {
        setStatus('detailStatus', error.message, 'error');
    }
}

function buildFilterParams() {
    const params = new URLSearchParams();
    addParam(params, 'phone', 'filterPhone');
    addParam(params, 'customer_id', 'filterCustomer');
    addParam(params, 'delivery_service', 'filterDelivery');
    addParam(params, 'bank', 'filterBank');
    addParam(params, 'currency', 'filterCurrency');
    addParam(params, 'city', 'filterCity');
    addParam(params, 'region', 'filterRegion');
    addParam(params, 'shardkey', 'filterShardkey');
    addParam(params, 'limit', 'filterLimit');
    return params.toString();
}

function addParam(params, name, elementId) {
    const value = document.getElementById(elementId).value.trim();
    if (value) {
        params.set(name, value);
    }
}

function fillSelect(elementId, values) {
    const select = document.getElementById(elementId);
    const currentValue = select.value;
    select.innerHTML = '<option value="">Все</option>';
    (values || []).forEach((value) => {
        const option = document.createElement('option');
        option.value = value;
        option.textContent = value;
        select.appendChild(option);
    });
    select.value = currentValue;
}

function renderOrders(orders) {
    const body = document.getElementById('ordersBody');
    body.innerHTML = orders.map((order) => `
        <tr>
            <td>${escapeHtml(order.order_uid)}</td>
            <td>${escapeHtml(order.phone)}</td>
            <td>${escapeHtml(order.customer_id)}</td>
            <td>${escapeHtml(order.delivery_service)}</td>
            <td>${escapeHtml(order.bank)}</td>
            <td>${formatMoney(order.amount, order.currency)}</td>
            <td>${escapeHtml(order.city)}</td>
            <td>${Number(order.items_count || 0)}</td>
            <td><button class="secondary" type="button" onclick="loadOrderDetail('${escapeAttr(order.order_uid)}')">Открыть</button></td>
        </tr>
    `).join('');
}

function renderAggregations(items) {
    const body = document.getElementById('aggregateBody');
    body.innerHTML = items.map((item) => `
        <tr>
            <td>${escapeHtml(item.key)}</td>
            <td>${Number(item.orders || 0)}</td>
            <td>${Number(item.total || 0)}</td>
            <td>${Number(item.average || 0).toFixed(2)}</td>
            <td>${Number(item.items_count || 0)}</td>
        </tr>
    `).join('');
}

function renderOrderDetail(order) {
    document.getElementById('detailOrderUid').value = order.order_uid;
    const dateCreated = new Date(order.date_created).toLocaleString();
    const items = order.items || [];

    document.getElementById('detailResult').innerHTML = `
        <div class="order-detail">
            <div class="detail-block">
                <h3>Заказ</h3>
                <p><strong>UID:</strong> ${escapeHtml(order.order_uid)}</p>
                <p><strong>Трек:</strong> ${escapeHtml(order.track_number)}</p>
                <p><strong>Клиент:</strong> ${escapeHtml(order.customer_id)}</p>
                <p><strong>Дата:</strong> ${escapeHtml(dateCreated)}</p>
                <p><strong>Shardkey:</strong> ${escapeHtml(order.shardkey)}</p>
            </div>
            <div class="detail-block">
                <h3>Доставка</h3>
                <p><strong>Получатель:</strong> ${escapeHtml(order.delivery.name)}</p>
                <p><strong>Телефон:</strong> ${escapeHtml(order.delivery.phone)}</p>
                <p><strong>Город:</strong> ${escapeHtml(order.delivery.city)}</p>
                <p><strong>Регион:</strong> ${escapeHtml(order.delivery.region)}</p>
                <p><strong>Email:</strong> ${escapeHtml(order.delivery.email)}</p>
            </div>
            <div class="detail-block">
                <h3>Оплата</h3>
                <p><strong>Сумма:</strong> ${formatMoney(order.payment.amount, order.payment.currency)}</p>
                <p><strong>Провайдер:</strong> ${escapeHtml(order.payment.provider)}</p>
                <p><strong>Банк:</strong> ${escapeHtml(order.payment.bank)}</p>
                <p><strong>Товары:</strong> ${Number(order.payment.goods_total || 0)}</p>
            </div>
        </div>
        <h3 style="margin-top:16px">Товары</h3>
        <div class="items-grid">
            ${items.map((item) => `
                <div class="item-card">
                    <p><strong>${escapeHtml(item.name)}</strong></p>
                    <p>Цена: ${Number(item.price || 0)}</p>
                    <p>Бренд: ${escapeHtml(item.brand)}</p>
                    <p>Артикул: ${Number(item.chrt_id || 0)}</p>
                    <p>Статус: ${Number(item.status || 0)}</p>
                </div>
            `).join('')}
        </div>
    `;
}

async function apiFetch(url, options = {}) {
    const response = await fetch(url, {
        ...options,
        headers: {
            ...(options.headers || {}),
            Authorization: `Bearer ${state.token}`,
        },
    });

    const data = await readJSON(response);
    if (!response.ok) {
        throw new Error(data.error || `HTTP ${response.status}`);
    }
    return data;
}

async function readJSON(response) {
    const text = await response.text();
    if (!text) {
        return {};
    }
    try {
        return JSON.parse(text);
    } catch (error) {
        return {error: text};
    }
}

async function refreshDashboard() {
    try {
        await loadFilterValues();
    } catch (error) {
        setStatus('ordersStatus', `Фильтры не загрузились: ${error.message}`, 'error');
    }

    await loadOrders();
    await loadAggregate();
}

function setStatus(elementId, message, type = '') {
    const element = document.getElementById(elementId);
    element.textContent = message;
    element.className = `status ${type}`.trim();
}

function formatMoney(amount, currency) {
    return `${Number(amount || 0)} ${escapeHtml(currency || '')}`;
}

function escapeHtml(value) {
    return String(value ?? '')
        .replace(/&/g, '&amp;')
        .replace(/</g, '&lt;')
        .replace(/>/g, '&gt;')
        .replace(/"/g, '&quot;')
        .replace(/'/g, '&#039;');
}

function escapeAttr(value) {
    return escapeHtml(value).replace(/`/g, '&#096;');
}
