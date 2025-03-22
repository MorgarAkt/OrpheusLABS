/** @type {import('tailwindcss').Config} */
module.exports = {
    tailwindcss: {},
    autoprefixer: {},
    content: ["./**/*.{html,js}",'!./node_modules/**/*',],
    theme: {
        extend: {
            colors: {
                custom: {
                    primary: '#F55B47',
                    secondary: '#CCF24E',
                    text: '#44403c',
                }
            },
            fontFamily: {
                newamsterdam: ["New Amsterdam", "serif"],
            },
        },
    },
    corePlugins: {
        float: false,
        objectFit: false,
    },
    plugins: [
        require('autoprefixer'),
        require('postcss-preset-env'),
      ]
};
