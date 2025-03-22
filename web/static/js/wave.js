const canvas = document.getElementById('waveCanvas');
const ctx = canvas.getContext('2d');
let width, height;

function resize() {
    width = canvas.width = window.innerWidth;
    height = canvas.height = window.innerHeight;
}
window.addEventListener('resize', resize);
resize();

let t = 0;
let speedMultiplier = 1;

function setBPM(bpm) {
    speedMultiplier = bpm / 60;
}
setBPM(60);

function animate() {
    t += 0.02 * speedMultiplier;
    ctx.clearRect(0, 0, width, height);

    for (let i = 0; i < 3; i++) {
        ctx.beginPath();
        const offset = i * 105;
        ctx.moveTo(0, height / 2 + offset);
        for (let x = 0; x < width; x += 15) {
            const y = height / 2 + offset + Math.sin(x * 0.04 + t + offset * 0.04) * 20;
            ctx.lineTo(x, y);
        }
        ctx.lineTo(width, height);
        ctx.lineTo(0, height);
        ctx.closePath();

        ctx.fillStyle = i === 0 ? "#CCF547" : i === 1 ? "#F55B47" : "#CCF547"; 
        ctx.globalAlpha = 1;
        ctx.fill();
    }
    requestAnimationFrame(animate);
}
animate();

window.setBPM = setBPM;