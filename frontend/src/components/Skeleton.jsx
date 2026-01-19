import React from 'react';

export const Skeleton = ({ className = '', variant = 'rectangular' }) => {
    const baseClasses = 'animate-pulse bg-gray-200';
    const variantClasses = {
        rectangular: 'rounded',
        circular: 'rounded-full',
        text: 'rounded h-4',
    };

    return (
        <div className={`${baseClasses} ${variantClasses[variant]} ${className}`} />
    );
};

export const StatCardSkeleton = () => (
    <div className="bg-white overflow-hidden shadow rounded-lg p-5">
        <div className="flex items-center">
            <div className="flex-shrink-0">
                <Skeleton variant="circular" className="h-12 w-12" />
            </div>
            <div className="ml-5 w-0 flex-1">
                <Skeleton className="h-4 w-24 mb-2" />
                <Skeleton className="h-8 w-16" />
            </div>
        </div>
    </div>
);

export const CardSkeleton = () => (
    <div className="bg-white shadow rounded-lg p-6">
        <Skeleton className="h-6 w-48 mb-4" />
        <div className="space-y-3">
            <Skeleton className="h-4 w-full" />
            <Skeleton className="h-4 w-5/6" />
            <Skeleton className="h-4 w-4/6" />
        </div>
    </div>
);