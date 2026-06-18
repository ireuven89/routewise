import { BrowserRouter as Router, Routes, Route, Navigate } from 'react-router-dom';
import { AuthProvider } from './context/AuthContext';
import PrivateRoute from './components/PrivateRoute';
import Login from './pages/Login';
import Register from './pages/Register';
import Dashboard from './pages/Dashboard';
import Jobs from "./pages/Jobs";
import Customers from "./pages/Customers";
import Workers from "./pages/Workers";
import FindService from "./pages/FindService";
import OrganizationSettings from "./pages/OrganizationSettings";
import './rtl.css';
import {LanguageProvider} from "./context/LanguageContext";

function App() {
    return (
        <Router>
            <LanguageProvider>
            <AuthProvider>
                <Routes>
                    <Route path="/login" element={<Login />} />
                    <Route path="/register" element={<Register />} />

                    <Route
                        path="/dashboard"
                        element={
                            <PrivateRoute>
                                <Dashboard />
                            </PrivateRoute>
                        }
                    />

                    {/* Placeholder routes - we'll build these next */}
                    <Route
                        path="/jobs"
                        element={
                            <PrivateRoute>
                                <Jobs />
                            </PrivateRoute>
                        }
                    />
                    <Route
                        path="/customers"
                        element={
                            <PrivateRoute>
                                <Customers />
                            </PrivateRoute>
                        }
                    />
                    <Route
                        path="/technicians"
                        element={
                            <PrivateRoute>
                                <Workers />
                            </PrivateRoute>
                        }
                    />

                    <Route
                        path="/settings"
                        element={
                            <PrivateRoute>
                                <OrganizationSettings />
                            </PrivateRoute>
                        }
                    />

                    {/* Public customer discovery page */}
                    <Route path="/find-service" element={<FindService />} />

                    <Route path="/" element={<Navigate to="/dashboard" replace />} />
                </Routes>
            </AuthProvider>
            </LanguageProvider>
        </Router>
    );
}

export default App;