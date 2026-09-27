import React, { useEffect, useMemo, useState } from 'react';
import { NavigationContainer } from '@react-navigation/native';
import { createNativeStackNavigator } from '@react-navigation/native-stack';
import AsyncStorage from '@react-native-async-storage/async-storage';
import { storage } from '../services/api';
import { ModeContext } from './ModeContext';
import RoleSelectScreen from '../screens/customer/RoleSelectScreen';
import FindServiceScreen from '../screens/customer/FindServiceScreen';
import ProviderResultsScreen from '../screens/customer/ProviderResultsScreen';
import PostJobScreen from '../screens/customer/PostJobScreen';
import RequestTrackingScreen from '../screens/customer/RequestTrackingScreen';
import MyRequestsScreen from '../screens/customer/MyRequestsScreen';
import WorkerLoginScreen from '../screens/auth/WorkerLoginScreen';
import ProjectsListScreen from '../screens/projects/ProjectsListScreen';
import ProjectDetailScreen from '../screens/projects/ProjectDetailScreen';

const Stack = createNativeStackNavigator();

// Root: RoleSelect until the user picks a mode (remembered on the device).
//   customer   → public find-service flow, no login
//   technician → existing OTP login → jobs
const AppNavigator = () => {
  const [mode, setModeState] = useState(null);
  const [isLoggedIn, setIsLoggedIn] = useState(false);
  const [isLoading, setIsLoading] = useState(true);

  useEffect(() => {
    storage.getMode().then((saved) => {
      setModeState(saved);
      setIsLoading(false);
    });
  }, []);

  // Technician login/logout write the token straight to storage, so keep watching it.
  useEffect(() => {
    if (mode !== 'technician') return undefined;
    const check = async () => setIsLoggedIn(!!(await AsyncStorage.getItem('token')));
    check();
    const interval = setInterval(check, 1000);
    return () => clearInterval(interval);
  }, [mode]);

  const modeValue = useMemo(() => ({
    mode,
    setMode: (next) => {
      setModeState(next);
      storage.setMode(next);
    },
  }), [mode]);

  if (isLoading) return null;

  let screens;
  if (mode === 'customer') {
    screens = (
      <>
        <Stack.Screen name="FindService" component={FindServiceScreen} />
        <Stack.Screen name="ProviderResults" component={ProviderResultsScreen} />
        <Stack.Screen name="PostJob" component={PostJobScreen} />
        <Stack.Screen name="RequestTracking" component={RequestTrackingScreen} />
        <Stack.Screen name="MyRequests" component={MyRequestsScreen} />
      </>
    );
  } else if (mode === 'technician') {
    screens = isLoggedIn ? (
      <>
        <Stack.Screen name="ProjectsList" component={ProjectsListScreen} />
        <Stack.Screen name="ProjectDetail" component={ProjectDetailScreen} options={{ headerShown: true, title: '' }} />
      </>
    ) : (
      <Stack.Screen name="Login" component={WorkerLoginScreen} />
    );
  } else {
    screens = <Stack.Screen name="RoleSelect" component={RoleSelectScreen} />;
  }

  return (
    <ModeContext.Provider value={modeValue}>
      <NavigationContainer>
        <Stack.Navigator screenOptions={{ headerShown: false }}>{screens}</Stack.Navigator>
      </NavigationContainer>
    </ModeContext.Provider>
  );
};

export default AppNavigator;
