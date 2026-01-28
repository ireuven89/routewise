import React, { useState, useEffect } from 'react';
import { Link } from 'react-router-dom';
import {
    FaBriefcase,
    FaCalendarCheck,
    FaUsers,
    FaUserCog,
    FaPlus,
    FaClock,
    FaCheckCircle,
    FaChartLine,
} from 'react-icons/fa';
import Layout from '../components/Layout';
import { StatCardSkeleton, CardSkeleton } from '../components/Skeleton';
import {customersAPI, jobsAPI, workersAPI} from '../api/client';

const Dashboard = () => {
    const [stats, setStats] = useState({
        totalJobs: 0,
        scheduledJobs: 0,
        totalCustomers: 0,
        totalWorkers: 0,
    });
    const [todayJobs, setTodayJobs] = useState([]);
    const [recentActivity, setRecentActivity] = useState([]);
    const [loading, setLoading] = useState(true);

    useEffect(() => {
        fetchDashboardData();
    }, []);

    const fetchDashboardData = async () => {
        setLoading(true);
        try {
            // Fetch all data in parallel
            const [jobsRes, customersRes, techniciansRes] = await Promise.all([
                jobsAPI.getAll(),
                customersAPI.getAll(),
                workersAPI.getAll(false),
            ]);

            const jobs = jobsRes.data || [];
            const scheduledJobs = jobs.filter((j) => j.status === 'scheduled');

            // Get today's jobs
            const todayStart = new Date();
            todayStart.setHours(0, 0, 0, 0);
            const todayEnd = new Date();
            todayEnd.setHours(23, 59, 59, 999);

            console.log("all jobs:", jobs.length)
            console.log("schdeuled jobs", scheduledJobs.length)
            const todayScheduled = jobs.filter((j) => {
                const jobDate = new Date(j.scheduled_at);
                console.log("todays date: ", jobDate)
                return jobDate >= todayStart && jobDate <= todayEnd;
            }).slice(0, 5);

            setStats({
                totalJobs: jobs.length,
                scheduledJobs: scheduledJobs.length,
                totalCustomers: customersRes.data?.length || 0,
                totalWorkers: techniciansRes.data?.length || 0,
            });

            setTodayJobs(todayScheduled);

            // Mock recent activity (replace with real API later)
            setRecentActivity([
                { id: 1, type: 'job_created', description: 'New job created for AC Repair', time: '10 mins ago' },
                { id: 2, type: 'job_completed', description: 'Heating System Maintenance completed', time: '1 hour ago' },
                { id: 3, type: 'customer_added', description: 'New customer added', time: '2 hours ago' },
                { id: 4, type: 'technician_assigned', description: 'Technician assigned to job', time: '3 hours ago' },
            ]);
        } catch (error) {
            console.error('Error fetching dashboard data:', error);
        } finally {
            setLoading(false);
        }
    };

    return (
        <Layout>
            <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
                {/* Header */}
                <div className="flex justify-between items-center mb-8">
                    <h1 className="text-3xl font-bold text-gray-900">Dashboard</h1>
                    <Link
                        to="/jobs"
                        className="inline-flex items-center px-4 py-2 bg-blue-600 hover:bg-blue-700 text-white font-medium rounded-lg transition-colors duration-200 shadow-sm hover:shadow-md"
                    >
                        <FaPlus className="mr-2" />
                        New Job
                    </Link>
                </div>

                {/* Stats Grid */}
                <div className="grid grid-cols-1 gap-5 sm:grid-cols-2 lg:grid-cols-4 mb-8">
                    {loading ? (
                        <>
                            <StatCardSkeleton />
                            <StatCardSkeleton />
                            <StatCardSkeleton />
                            <StatCardSkeleton />
                        </>
                    ) : (
                        <>
                            <StatCard
                                title="Total Jobs"
                                value={stats.totalJobs}
                                link="/jobs"
                                icon={FaBriefcase}
                                color="blue"
                                trend="+12%"
                            />
                            <StatCard
                                title="Scheduled Today"
                                value={stats.scheduledJobs}
                                link="/jobs?status=scheduled"
                                icon={FaCalendarCheck}
                                color="green"
                                trend="+8%"
                            />
                            <StatCard
                                title="Customers"
                                value={stats.totalCustomers}
                                link="/customers"
                                icon={FaUsers}
                                color="purple"
                                trend="+23%"
                            />
                            <StatCard
                                title="Technicians"
                                value={stats.totalWorkers}
                                link="/technicians"
                                icon={FaUserCog}
                                color="orange"
                                trend="+5%"
                            />
                        </>
                    )}
                </div>

                {/* Two Column Layout */}
                <div className="grid grid-cols-1 lg:grid-cols-3 gap-6 mb-8">
                    {/* Today's Schedule - Takes 2 columns */}
                    <div className="lg:col-span-2">
                        {loading ? (
                            <CardSkeleton />
                        ) : (
                            <div className="bg-white shadow rounded-lg overflow-hidden">
                                <div className="px-6 py-4 border-b border-gray-200 bg-gradient-to-r from-blue-50 to-blue-100">
                                    <div className="flex items-center justify-between">
                                        <h2 className="text-lg font-semibold text-gray-900 flex items-center">
                                            <FaClock className="mr-2 text-blue-600" />
                                            Today's Schedule
                                        </h2>
                                        <Link
                                            to="/jobs"
                                            className="text-sm text-blue-600 hover:text-blue-700 font-medium"
                                        >
                                            View All
                                        </Link>
                                    </div>
                                </div>
                                <div className="divide-y divide-gray-200">
                                    {todayJobs.length === 0 ? (
                                        <div className="px-6 py-12 text-center">
                                            <FaCalendarCheck className="mx-auto h-12 w-12 text-gray-400 mb-3" />
                                            <p className="text-gray-500 text-sm">No jobs scheduled for today</p>
                                            <Link
                                                to="/jobs"
                                                className="mt-4 inline-flex items-center text-sm text-blue-600 hover:text-blue-700 font-medium"
                                            >
                                                <FaPlus className="mr-1" />
                                                Schedule a job
                                            </Link>
                                        </div>
                                    ) : (
                                        todayJobs.map((job) => (
                                            <Link
                                                key={job.id}
                                                to={`/jobs/${job.id}`}
                                                className="block px-6 py-4 hover:bg-gray-50 transition-colors duration-150"
                                            >
                                                <div className="flex items-center justify-between">
                                                    <div className="flex-1">
                                                        <p className="text-sm font-medium text-gray-900">{job.title}</p>
                                                        <p className="text-sm text-gray-500 mt-1 line-clamp-1">{job.description}</p>
                                                        <div className="flex items-center mt-2 space-x-4">
                              <span className="inline-flex items-center text-xs text-gray-600">
                                <FaClock className="mr-1" />
                                  {new Date(job.scheduled_at).toLocaleTimeString('en-US', {
                                      hour: '2-digit',
                                      minute: '2-digit',
                                  })}
                              </span>
                                                            <StatusBadge status={job.status} />
                                                        </div>
                                                    </div>
                                                </div>
                                            </Link>
                                        ))
                                    )}
                                </div>
                            </div>
                        )}
                    </div>

                    {/* Recent Activity - Takes 1 column */}
                    <div className="lg:col-span-1">
                        {loading ? (
                            <CardSkeleton />
                        ) : (
                            <div className="bg-white shadow rounded-lg overflow-hidden">
                                <div className="px-6 py-4 border-b border-gray-200 bg-gradient-to-r from-purple-50 to-purple-100">
                                    <h2 className="text-lg font-semibold text-gray-900 flex items-center">
                                        <FaChartLine className="mr-2 text-purple-600" />
                                        Recent Activity
                                    </h2>
                                </div>
                                <div className="px-6 py-4">
                                    <div className="flow-root">
                                        <ul className="-mb-8">
                                            {recentActivity.map((activity, idx) => (
                                                <li key={activity.id}>
                                                    <div className="relative pb-8">
                                                        {idx !== recentActivity.length - 1 && (
                                                            <span
                                                                className="absolute top-5 left-5 -ml-px h-full w-0.5 bg-gray-200"
                                                                aria-hidden="true"
                                                            />
                                                        )}
                                                        <div className="relative flex items-start space-x-3">
                                                            <div>
                                                                <div className="relative px-1">
                                                                    <div className="h-8 w-8 bg-blue-100 rounded-full ring-8 ring-white flex items-center justify-center">
                                                                        <ActivityIcon type={activity.type} />
                                                                    </div>
                                                                </div>
                                                            </div>
                                                            <div className="min-w-0 flex-1">
                                                                <div>
                                                                    <p className="text-sm text-gray-700">{activity.description}</p>
                                                                    <p className="mt-0.5 text-xs text-gray-500">{activity.time}</p>
                                                                </div>
                                                            </div>
                                                        </div>
                                                    </div>
                                                </li>
                                            ))}
                                        </ul>
                                    </div>
                                </div>
                            </div>
                        )}
                    </div>
                </div>

                {/* Quick Actions */}
                <div className="bg-white shadow rounded-lg p-6">
                    <h2 className="text-lg font-semibold text-gray-900 mb-4 flex items-center">
                        <FaPlus className="mr-2 text-gray-600" />
                        Quick Actions
                    </h2>
                    <div className="grid grid-cols-1 sm:grid-cols-3 gap-4">
                        <QuickActionButton
                            to="/jobs"
                            icon={FaBriefcase}
                            text="Create New Job"
                            color="blue"
                        />
                        <QuickActionButton
                            to="/customers"
                            icon={FaUsers}
                            text="Add Customer"
                            color="green"
                        />
                        <QuickActionButton
                            to="/technicians"
                            icon={FaUserCog}
                            text="Add Technician"
                            color="purple"
                        />
                    </div>
                </div>
            </div>
        </Layout>
    );
};

// Stat Card Component
const StatCard = ({ title, value, link, icon: Icon, color, trend }) => {
    const colorClasses = {
        blue: 'text-blue-600',
        green: 'text-green-600',
        purple: 'text-purple-600',
        orange: 'text-orange-600',
    };

    const bgColorClasses = {
        blue: 'bg-blue-50',
        green: 'bg-green-50',
        purple: 'bg-purple-50',
        orange: 'bg-orange-50',
    };

    return (
        <Link
            to={link}
            className="bg-white overflow-hidden shadow rounded-lg hover:shadow-lg transition-all duration-200 transform hover:-translate-y-1"
        >
            <div className="p-5">
                <div className="flex items-center">
                    <div className="flex-shrink-0">
                        <div className={`${bgColorClasses[color]} rounded-md p-3`}>
                            <Icon className={`h-6 w-6 ${colorClasses[color]}`} />
                        </div>
                    </div>
                    <div className="ml-5 w-0 flex-1">
                        <dl>
                            <dt className="text-sm font-medium text-gray-500 truncate">{title}</dt>
                            <dd className="flex items-baseline">
                                <div className="text-2xl font-semibold text-gray-900">{value}</div>
                                {trend && (
                                    <div className="ml-2 flex items-baseline text-sm font-semibold text-green-600">
                                        <FaChartLine className="mr-1 h-3 w-3" />
                                        {trend}
                                    </div>
                                )}
                            </dd>
                        </dl>
                    </div>
                </div>
            </div>
        </Link>
    );
};

// Quick Action Button Component
const QuickActionButton = ({ to, icon: Icon, text, color }) => {
    const colorClasses = {
        blue: 'bg-blue-600 hover:bg-blue-700',
        green: 'bg-green-600 hover:bg-green-700',
        purple: 'bg-purple-600 hover:bg-purple-700',
    };

    return (
        <Link
            to={to}
            className={`${colorClasses[color]} text-white px-6 py-3 rounded-lg text-center font-medium transition-all duration-200 flex items-center justify-center shadow-sm hover:shadow-md`}
        >
            <Icon className="mr-2" />
            {text}
        </Link>
    );
};

// Status Badge Component
const StatusBadge = ({ status }) => {
    const statusConfig = {
        scheduled: { color: 'bg-blue-100 text-blue-800', text: 'Scheduled' },
        in_progress: { color: 'bg-yellow-100 text-yellow-800', text: 'In Progress' },
        completed: { color: 'bg-green-100 text-green-800', text: 'Completed' },
        cancelled: { color: 'bg-red-100 text-red-800', text: 'Cancelled' },
    };

    const config = statusConfig[status] || statusConfig.scheduled;

    return (
        <span
            className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${config.color}`}
        >
      {config.text}
    </span>
    );
};

// Activity Icon Component
const ActivityIcon = ({ type }) => {
    const icons = {
        job_created: <FaPlus className="h-4 w-4 text-blue-600" />,
        job_completed: <FaCheckCircle className="h-4 w-4 text-green-600" />,
        customer_added: <FaUsers className="h-4 w-4 text-purple-600" />,
        technician_assigned: <FaUserCog className="h-4 w-4 text-orange-600" />,
    };

    return icons[type] || <FaBriefcase className="h-4 w-4 text-gray-600" />;
};

export default Dashboard;