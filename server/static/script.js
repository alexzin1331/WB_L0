async function getOrder() {
    const orderId = document.getElementById('orderId').value.trim();
    if (!orderId) {
        showResult('error', 'Пожалуйста, введите ID заказа');
        return;
    }

    showResult('loading', 'Загрузка данных...');

    try {
        const response = await fetch(`/order/${orderId}`);
        if (!response.ok) {
            throw new Error(`Ошибка: ${response.status}`);
        }

        const order = await response.json();
        renderOrder(order);
    } catch (error) {
        console.error('Ошибка:', error);
        showResult('error', `Не удалось загрузить заказ: ${error.message}`);
    }
}

function renderOrder(order) {
    const resultDiv = document.getElementById('result');

    // Форматирование даты
    const dateCreated = new Date(order.date_created).toLocaleString();

    let html = `
        <div class="order-section">
            <h2>Заказ #${order.order_uid}</h2>
            <p><strong>Трек-номер:</strong> ${order.track_number}</p>
            <p><strong>Дата создания:</strong> ${dateCreated}</p>
            <p><strong>Клиент:</strong> ${order.customer_id}</p>
        </div>
        
        <div class="order-section">
            <h3>Доставка</h3>
            <p><strong>Получатель:</strong> ${order.delivery.name}</p>
            <p><strong>Телефон:</strong> ${order.delivery.phone}</p>
            <p><strong>Адрес:</strong> ${order.delivery.city}, ${order.delivery.address}</p>
            <p><strong>Email:</strong> ${order.delivery.email}</p>
        </div>
        
        <div class="order-section">
            <h3>Оплата</h3>
            <p><strong>Сумма:</strong> ${order.payment.amount} ${order.payment.currency}</p>
            <p><strong>Провайдер:</strong> ${order.payment.provider}</p>
            <p><strong>Банк:</strong> ${order.payment.bank}</p>
        </div>
        
        <div class="order-section">
            <h3>Товары (${order.items.length})</h3>
            <div class="items-grid">
    `;

    order.items.forEach(item => {
        html += `
            <div class="item-card">
                <p><strong>${item.name}</strong></p>
                <p>Цена: $${item.price}</p>
                <p>Бренд: ${item.brand}</p>
                <p>Артикул: ${item.chrt_id}</p>
            </div>
        `;
    });

    html += `
            </div>
        </div>
    `;

    resultDiv.innerHTML = html;
}

function showResult(type, message) {
    const resultDiv = document.getElementById('result');

    if (type === 'loading') {
        resultDiv.innerHTML = `<div class="result">${message}</div>`;
    }
    else if (type === 'error') {
        resultDiv.innerHTML = `<div class="error">${message}</div>`;
    }
}