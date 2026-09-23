// jest-dom adds custom jest matchers for asserting on DOM nodes.
// allows you to do things like:
// expect(element).toHaveTextContent(/react/i)
// learn more: https://github.com/testing-library/jest-dom
import '@testing-library/jest-dom';

// react-router v7 uses TextEncoder/TextDecoder, which jsdom (Jest 27 via react-scripts)
// doesn't provide.
import { TextEncoder, TextDecoder } from 'util';

if (typeof global.TextEncoder === 'undefined') {
    global.TextEncoder = TextEncoder;
    global.TextDecoder = TextDecoder;
}
