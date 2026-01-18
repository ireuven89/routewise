import { useState, useEffect } from 'react';
import { Link } from 'react-router-dom';
import { jobsAPI, customersAPI, techniciansAPI } from '../api/client';
import Layout from '../components/Layout';

const Dashboard = () => {
    const [stats, setStats] = useState({
        totalJobs: 0,
        totalCustomers: 0,
        totalTechnicians: 0,
        scheduledJobs: 0,
        loading: true,
    });

    useEffect(() => {
        loadStats();
    }, []);

    const loadStats = async () => {
        try {
            const [jobsRes, customersRes, techniciansRes] = await Promise.all([
                jobsAPI.getAll(),
                customersAPI.getAll(),
                techniciansAPI.getAll(false),
            ]);

            const jobs = jobsRes.data || [];
            const scheduledJobs = jobs.filter(j => j.status === 'scheduled').length;

            setStats({
                totalJobs: jobs.length,
                totalCustomers: customersRes.data?.length || 0,
                totalTechnicians: techniciansRes.data?.length || 0,
                scheduledJobs,
                loading: false,
            });
        } catch (error) {
            console.error('Failed to load stats:', error);
            setStats(prev => ({ ...prev, loading: false }));
        }
    };

    if (stats.loading) {
        return (
            <Layout>
                <div className="flex justify-center items-center h-64">
                    <div className="text-lg text-gray-600">Loading...</div>
                </div>
            </Layout>
        );
    }

    return (
        <Layout>
            <div className="px-4 sm:px-0">
                <h1 className="text-3xl font-bold text-gray-900 mb-8">Dashboard</h1>

                {/* Stats Grid */}
                <div className="grid grid-cols-1 gap-5 sm:grid-cols-2 lg:grid-cols-4 mb-8">
                    <StatCard
                        title="Total Jobs"
                        value={stats.totalJobs}
                        link="/jobs"
                        color="blue"
                    />
                    <StatCard
                        title="Scheduled Jobs"
                        value={stats.scheduledJobs}
                        link="/jobs?status=scheduled"
                        color="green"
                    />
                    <StatCard
                        title="Customers"
                        value={stats.totalCustomers}
                        link="/customers"
                        color="purple"
                    />
                    <StatCard
                        title="Technicians"
                        value={stats.totalTechnicians}
                        link="/technicians"
                        color="orange"
                    />
                </div>

                {/* Quick Actions */}
                <div className="bg-white shadow rounded-lg p-6">
                    <h2 className="text-lg font-medium text-gray-900 mb-4">Quick Actions</h2>
                    <div className="grid grid-cols-1 sm:grid-cols-3 gap-4">
                        <Link
                            to="/jobs/new"
                            className="bg-blue-600 hover:bg-blue-700 text-white px-6 py-3 rounded-md text-center font-medium"
                        >
                            Create New Job
                        </Link>
                        <Link
                            to="/customers/new"
                            className="bg-green-600 hover:bg-green-700 text-white px-6 py-3 rounded-md text-center font-medium"
                        >
                            Add Customer
                        </Link>
                        <Link
                            to="/technicians/new"
                            className="bg-purple-600 hover:bg-purple-700 text-white px-6 py-3 rounded-md text-center font-medium"
                        >
                            Add Technician
                        </Link>
                    </div>
                </div>
            </div>
        </Layout>
    );
};

const StatCard = ({ title, value, link, color }) => {
    const colorClasses = {
        blue: 'bg-blue-50 text-blue-700',
        green: 'bg-green-50 text-green-700',
        purple: 'bg-purple-50 text-purple-700',
        orange: 'bg-orange-50 text-orange-700',
    };

    return (
        <Link to={link} className="bg-white overflow-hidden shadow rounded-lg hover:shadow-md transition-shadow">
            <div className="px-4 py-5 sm:p-6">
                <dt className="text-sm font-medium text-gray-500 truncate">{title}</dt>
                <dd className={`mt-1 text-3xl font-semibold ${colorClasses[color]}`}>
                    {value}
                </dd>
            </div>
        </Link>
    );
};

export default Dashboard;