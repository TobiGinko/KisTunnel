// 获取DOM元素
const vkeyInput = document.getElementById('vkey');
const addressInput = document.getElementById('address');
const portInput = document.getElementById('port');
const connectBtn = document.getElementById('connectBtn');
const disconnectBtn = document.getElementById('disconnectBtn');
const statusElement = document.getElementById('status');
const customAlert = document.getElementById('customAlert');
const customAlertMessage = document.getElementById('customAlertMessage');
const customAlertClose = document.getElementById('customAlertClose');

// 自定义弹窗函数
function showCustomAlert(message) {
    console.log('显示提示:', message);
    customAlertMessage.textContent = message;
    customAlert.style.display = 'flex';
}

// 关闭自定义弹窗
customAlertClose.addEventListener('click', () => {
    customAlert.style.display = 'none';
});

// 验证输入
function validateInputs() {
    const vkey = vkeyInput.value.trim();
    const address = addressInput.value.trim();
    const port = portInput.value.trim();

    if (!vkey) {
        showCustomAlert('请输入密钥');
        vkeyInput.focus();
        return false;
    }
    if (!address) {
        showCustomAlert('请输入连接地址');
        addressInput.focus();
        return false;
    }
    if (!port) {
        showCustomAlert('请输入端口');
        portInput.focus();
        return false;
    }

    // 验证端口号格式
    const portNum = parseInt(port);
    if (isNaN(portNum) || portNum < 1 || portNum > 65535) {
        showCustomAlert('端口号必须是1-65535之间的数字');
        portInput.focus();
        return false;
    }

    return true;
}

// 更新状态显示
function updateStatus(status, ip) {
    statusElement.textContent = status;
    
    if (status === '已连接') {
        statusElement.classList.add('connected');
    } else {
        statusElement.classList.remove('connected');
    }
}

// 更新按钮状态
function updateButtonState(isConnected) {
    connectBtn.disabled = isConnected;
    disconnectBtn.disabled = !isConnected;
}

// 保存配置到本地存储
function saveConfig() {
    const config = {
        vkey: vkeyInput.value.trim(),
        address: addressInput.value.trim(),
        port: portInput.value.trim()
    };
    localStorage.setItem('npc-config', JSON.stringify(config));
}

// 从本地存储加载配置
function loadConfig() {
    const configStr = localStorage.getItem('npc-config');
    if (configStr) {
        try {
            const config = JSON.parse(configStr);
            vkeyInput.value = config.vkey || '';
            addressInput.value = config.address || '';
            portInput.value = config.port || '';
        } catch (error) {
            console.error('加载配置失败:', error);
        }
    }
}

// 初始化
async function init() {
    // 加载保存的配置
    loadConfig();

    // 监听状态更新事件
    window.runtime.EventsOn("status-update", (status, ip) => {
        console.log('状态更新:', { status, ip });
        updateStatus(status, ip);
    });

    // 监听连接状态事件
    window.runtime.EventsOn("connection-state", (isConnected) => {
        console.log('连接状态更新:', isConnected);
        updateButtonState(isConnected);
    });

    // 监听auth-error事件
    window.runtime.EventsOn("auth-error", (msg) => {
        showCustomAlert(msg);
    });

    // 获取当前状态
    const [status, ip] = await window.go.main.App.GetStatus();
    console.log('当前状态:', { status, ip });
    updateStatus(status, ip);
    
    // 根据状态设置按钮状态
    updateButtonState(status === '已连接');
}

// 连接按钮点击事件
connectBtn.addEventListener('click', async () => {
    if (!validateInputs()) {
        return;
    }

    try {
        // 保存配置
        saveConfig();
        
        await window.go.main.App.Connect(
            addressInput.value.trim(),
            portInput.value.trim(),
            vkeyInput.value.trim()
        );
    } catch (error) {
        console.error('连接失败:', error);
        updateButtonState(false);
    }
});

// 断开按钮点击事件
disconnectBtn.addEventListener('click', async () => {
    try {
        await window.go.main.App.Disconnect();
    } catch (error) {
        console.error('断开连接失败:', error);
        showCustomAlert(error.message || '断开连接失败');
        updateButtonState(true);
    }
});

// 初始化应用
init(); 